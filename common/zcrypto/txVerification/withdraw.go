/*
 * Copyright © 2021 Zecrey Protocol
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package txVerification

import (
	"errors"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/zecrey-labs/zecrey-crypto/ffmath"
	"github.com/zecrey-labs/zecrey-crypto/wasm/zecrey-legend/legendTxTypes"
	"github.com/zecrey-labs/zecrey-legend/common/model/account"
	"github.com/zecrey-labs/zecrey-legend/common/model/asset"
	"github.com/zeromicro/go-zero/core/logx"
	"log"
	"math/big"
)

/*
	VerifyWithdrawTx:
	accounts order is:
	- FromAccount
		- Assets:
			- AssetA
			- AssetGas
	- GasAccount
		- Assets:
			- AssetGas
*/
func VerifyWithdrawTxInfo(
	accountInfoMap map[int64]*account.Account,
	assetInfoMap map[int64]map[int64]*asset.AccountAsset,
	txInfo *WithdrawTxInfo,
) (txDetails []*MempoolTxDetail, err error) {
	// verify params
	if accountInfoMap[txInfo.FromAccountIndex] == nil ||
		accountInfoMap[txInfo.GasAccountIndex] == nil ||
		assetInfoMap[txInfo.FromAccountIndex] == nil ||
		assetInfoMap[txInfo.FromAccountIndex][txInfo.AssetId] == nil ||
		assetInfoMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId] == nil ||
		assetInfoMap[txInfo.GasAccountIndex] == nil ||
		assetInfoMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId] == nil ||
		txInfo.AssetAmount.Cmp(ZeroBigInt) < 0 ||
		txInfo.GasFeeAssetAmount.Cmp(ZeroBigInt) < 0 {
		logx.Errorf("[VerifyTransferNftTxInfo] invalid params")
		return nil, errors.New("[VerifyTransferNftTxInfo] invalid params")
	}
	// verify nonce
	if txInfo.Nonce != accountInfoMap[txInfo.FromAccountIndex].Nonce {
		log.Println("[VerifyWithdrawTxInfo] invalid nonce")
		return nil, errors.New("[VerifyWithdrawTxInfo] invalid nonce")
	}
	var (
		assetDeltaMap = make(map[int64]map[int64]*big.Int)
	)
	// init delta map
	assetDeltaMap[txInfo.FromAccountIndex] = make(map[int64]*big.Int)
	if assetDeltaMap[txInfo.GasAccountIndex] == nil {
		assetDeltaMap[txInfo.GasAccountIndex] = make(map[int64]*big.Int)
	}
	// from account asset A
	assetDeltaMap[txInfo.FromAccountIndex][txInfo.AssetId] = ffmath.Neg(txInfo.AssetAmount)
	// from account asset Gas
	if assetDeltaMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId] == nil {
		assetDeltaMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId] = ffmath.Neg(txInfo.GasFeeAssetAmount)
	} else {
		assetDeltaMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId] = ffmath.Sub(
			assetDeltaMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId],
			txInfo.GasFeeAssetAmount,
		)
	}
	// gas account asset Gas
	if assetDeltaMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId] == nil {
		assetDeltaMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId] = txInfo.GasFeeAssetAmount
	} else {
		assetDeltaMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId] = ffmath.Add(
			assetDeltaMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId],
			txInfo.GasFeeAssetAmount,
		)
	}
	// check balance

	// compute hash
	hFunc := mimc.NewMiMC()
	msgHash := legendTxTypes.ComputeWithdrawMsgHash(txInfo, hFunc)
	// verify signature
	hFunc.Reset()
	pk, err := ParsePkStr(accountInfoMap[txInfo.FromAccountIndex].PublicKey)
	if err != nil {
		return nil, err
	}
	isValid, err := pk.Verify(txInfo.Sig, msgHash, hFunc)
	if err != nil {
		log.Println("[VerifyWithdrawTxInfo] unable to verify signature:", err)
		return nil, err
	}
	if !isValid {
		log.Println("[VerifyWithdrawTxInfo] invalid signature")
		return nil, errors.New("[VerifyWithdrawTxInfo] invalid signature")
	}
	// compute tx details
	// from account asset A
	txDetails = append(txDetails, &MempoolTxDetail{
		AssetId:      txInfo.AssetId,
		AssetType:    GeneralAssetType,
		AccountIndex: txInfo.FromAccountIndex,
		AccountName:  accountInfoMap[txInfo.FromAccountIndex].AccountName,
		Balance:      assetInfoMap[txInfo.FromAccountIndex][txInfo.AssetId].Balance,
		BalanceDelta: assetDeltaMap[txInfo.FromAccountIndex][txInfo.AssetId].String(),
	})
	// from account asset gas
	txDetails = append(txDetails, &MempoolTxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    GeneralAssetType,
		AccountIndex: txInfo.FromAccountIndex,
		AccountName:  accountInfoMap[txInfo.FromAccountIndex].AccountName,
		Balance:      assetInfoMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId].Balance,
		BalanceDelta: assetDeltaMap[txInfo.FromAccountIndex][txInfo.GasFeeAssetId].String(),
	})
	// gas account asset gas
	txDetails = append(txDetails, &MempoolTxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    GeneralAssetType,
		AccountIndex: txInfo.GasAccountIndex,
		AccountName:  accountInfoMap[txInfo.GasAccountIndex].AccountName,
		Balance:      assetInfoMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId].Balance,
		BalanceDelta: assetDeltaMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId].String(),
	})
	return txDetails, nil
}
