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
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
)

var secret = "mlcloud_systemcore"

// GenerateChallenge - Generate a challenge key based on md5
func GenerateChallenge(id string) string {
	hash := md5.New()
	return hex.EncodeToString(hash.Sum([]byte(id)))
}

// AuthenticateChallengeAnswer - Authenticate the challenge answer
func AuthenticateChallengeAnswer(id string, answer string) bool {
	hash := hmac.New(sha1.New, []byte(secret))
	effectiveID := GenerateChallenge(id)
	return (answer == hex.EncodeToString(hash.Sum([]byte(effectiveID))))
}
