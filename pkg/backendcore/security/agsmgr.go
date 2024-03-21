//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2020 Intel Corporation. All Rights Reserved.
//
//

package security

import (
	"2dacecommon/pkg/foundationcore"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"
)

var ldapPort string                       // Port number used by LDAP. By default it is 389 for non-TLS connection
var ldapServerList map[string][]string    // Per region server list, for some reason intel ldap server only contains per region users (amr/ger/gar/ccr)
var ldapUser string                       // our sys usr
var ldapPwd string                        // our sys pwd
var mlcloudEntitlementList map[int]string // our table of entitlement with key as 0 as highest entitlement. Note key must be 0 ... n with no gap in
// between

func init() {

	var err error

	// Read from sys_conf.json and populate variables
	ldapServerList = make(map[string][]string)
	readObjects := foundationcore.GetSystemConfig().GetConfig(foundationcore.ConfigType_LDAP).(map[string]interface{})

	regionLdapServerMap := readObjects["server_list"].(map[string]interface{})
	for key, val := range regionLdapServerMap {
		regionLdapServers := make([]string, 0)
		regionLdapServers = append(regionLdapServers, strings.Split(val.(string), ",")...)
		/*
			// Old code that is replaced by simplication
			for _, s := range strings.Split(val.(string), ",") {
				regionLdapServers = append(regionLdapServers, s)
			}
		*/
		ldapServerList[key] = regionLdapServers
	}

	ldapUser = readObjects["bound_usr"].(string)
	encryptedLdapPwd := readObjects["bound_pwd"].(string)
	ldapPwd, err = Decrypt(encryptedLdapPwd)
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovering panic retrieving AGS entitlement from LDAP. Error:", err)
		}
	}()

	if err != nil {
		fmt.Println("Failed on decrypting configured value")
		panic(err)
	}
	ldapPort = readObjects["port"].(string)
	mlcloudEntitlementString := readObjects["mlcloud_ags_entitlement"].([]interface{})
	mlcloudEntitlementList = make(map[int]string)
	for _, s := range mlcloudEntitlementString {
		tokens := strings.Split(s.(string), ":")
		if len(tokens) != 2 {
			panic(errors.New("invalid ntitlement entry in configuration"))
		}
		priorityInt, err := strconv.Atoi(tokens[0])
		if err != nil {
			panic(err)
		}
		mlcloudEntitlementList[priorityInt] = tokens[1]
	}
}

// AGSControlMgr is the entity that will manages all aspects of items related AGS business.
type AGSControlMgr struct {
}

// robustConnect - a robust ldap connect mechanism, very simple multiple ldap server list to try
func (o *AGSControlMgr) robustConnect(region string) (l *ldap.Conn, err error) {
	regionLdapServerList := ldapServerList[region]
	l = nil
	err = nil
	for _, ldapSvr := range regionLdapServerList {
		ldapSvrURL := fmt.Sprint("ldap://", ldapSvr, ":", ldapPort)
		l, err = ldap.DialURL(ldapSvrURL)
		if err == nil {
			break
		}
	}
	return l, err
}

// verifyUserId - Ensure the user id are in the format of domain\user, split it and return as tokens
func (o *AGSControlMgr) verifyUserID(uid string) (string, string, error) {
	tokens := strings.Split(uid, "\\")
	if len(tokens) == 2 {
		if len(tokens[0]) > 0 && len(tokens[1]) > 0 {
			return tokens[0], tokens[1], nil
		}
	}
	return "", "", errors.New("Uid must be in domain\\user format")
}

// match - Take a given entitlement and match against system configured list of entitlement, will return the high level entitlement
func (o *AGSControlMgr) match(entitlementToMatch string) (bool, string, int) {
	entryCnt := len(mlcloudEntitlementList)
	for i := 0; i < entryCnt; i++ {
		if strings.Contains(entitlementToMatch, mlcloudEntitlementList[i]) {
			return true, mlcloudEntitlementList[i], i
		}
	}
	return false, "", -1
}

// AuthorizationCheck - Authorization check against entitlement, will return highest entitlement if authorized
func (o *AGSControlMgr) AuthorizationCheck(uid string) (authorizedOk bool, highestLvlEntitlement string, err error) {

	// verify the format of uid and splits it into domain and user
	domain, uid, err := o.verifyUserID(uid)
	if err != nil {
		return authorizedOk, highestLvlEntitlement, err
	}

	ldapConn, err := o.robustConnect(domain)
	if err != nil {
		return authorizedOk, highestLvlEntitlement, err
	}
	defer ldapConn.Close()

	// We need to bind the connection to a valid domain user for the search ability to be enabled
	err = ldapConn.Bind(ldapUser, ldapPwd)
	if err != nil {
		return authorizedOk, highestLvlEntitlement, err
	}

	// construct our search, we are search for the user record on its memberOf attribute
	// if the memberOf attribute has entry our mlcloud entitlement, it is considered authorized
	searchReq := ldap.NewSearchRequest(
		fmt.Sprint("OU=Workers,DC=", domain, ",DC=corp,DC=intel,DC=com"),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprint("(sAMAccountName=", uid, ")"),
		[]string{"memberOf"},
		nil,
	)
	sr, err := ldapConn.Search(searchReq)
	if err != nil {
		return authorizedOk, highestLvlEntitlement, err
	}

	matchedEntitlements := make(map[int]string)
	entitlementLvls := make([]int, 0)
	for _, entry := range sr.Entries {
		for _, group := range entry.GetAttributeValues("memberOf") {
			authorizedOk, entitlement, level := o.match(group)
			if authorizedOk {
				matchedEntitlements[level] = entitlement
				entitlementLvls = append(entitlementLvls, level)
			}
		}
	}

	// find the highest entitlement from the list of matching entitlements
	if len(matchedEntitlements) > 0 {
		sort.Ints(entitlementLvls)
		highestLvlEntitlement = matchedEntitlements[entitlementLvls[0]]
		authorizedOk = true
	}
	highestLvlEntitlement = strings.ToUpper(highestLvlEntitlement)
	fmt.Println("highestLvlEntitlement =", highestLvlEntitlement)

	return authorizedOk, highestLvlEntitlement, err
}

func (o *AGSControlMgr) AuthorizeAndGetRoles(uid string) (authorizedOk bool, roleArry []string, emailStr string, err error) {

	// verify the format of uid and splits it into domain and user
	domain, uid, err := o.verifyUserID(uid)
	if err != nil {
		return authorizedOk, roleArry, emailStr, err
	}

	ldapConn, err := o.robustConnect(domain)
	if err != nil {
		return authorizedOk, roleArry, emailStr, err
	}
	defer ldapConn.Close()

	// We need to bind the connection to a valid domain user for the search ability to be enabled
	err = ldapConn.Bind(ldapUser, ldapPwd)
	if err != nil {
		return authorizedOk, roleArry, emailStr, err
	}

	// construct our search, we are search for the user record on its memberOf attribute
	// if the memberOf attribute has entry our mlcloud entitlement, it is considered authorized
	searchReq := ldap.NewSearchRequest(
		fmt.Sprint("OU=Workers,DC=", domain, ",DC=corp,DC=intel,DC=com"),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprint("(sAMAccountName=", uid, ")"),
		[]string{"memberOf", "sAMAccountName", "mail", "sn", "givenName", "cn"},
		nil,
	)
	sr, err := ldapConn.Search(searchReq)
	if err != nil {
		return authorizedOk, roleArry, emailStr, err
	}

	matchedEntitlements := make(map[int]string)
	entitlementLvls := make([]int, 0)
	for _, entry := range sr.Entries {
		for _, group := range entry.GetAttributeValues("memberOf") {
			authorizedOk, entitlement, level := o.match(group)
			if authorizedOk {
				matchedEntitlements[level] = entitlement
				entitlementLvls = append(entitlementLvls, level)
			}
		}
	}

	// find the highest entitlement from the list of matching entitlements
	if len(matchedEntitlements) > 0 {
		sort.Ints(entitlementLvls)
		//highestLvlEntitlement = matchedEntitlements[entitlementLvls[0]]
		authorizedOk = true
	}
	vals := sr.Entries[0].GetAttributeValues("memberOf")
	emailS := sr.Entries[0].GetAttributeValues("mail")

	emailStr = emailS[0]

	roleArry = make([]string, len(vals))
	for i, CN := range vals {
		roleArry[i] = CN[strings.Index(CN, "=")+1 : strings.Index(CN, ",")]
	}
	return authorizedOk, roleArry, emailStr, nil
}
