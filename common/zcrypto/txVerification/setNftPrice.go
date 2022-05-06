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
	"github.com/zecrey-labs/zecrey-core/common/general/model/nft"
	"github.com/zecrey-labs/zecrey-crypto/ffmath"
	"github.com/zecrey-labs/zecrey-crypto/wasm/zecrey-legend/legendTxTypes"
	"github.com/zecrey-labs/zecrey-legend/common/model/account"
	"github.com/zecrey-labs/zecrey-legend/common/model/asset"
	"github.com/zeromicro/go-zero/core/logx"
	"log"
	"math/big"
)

/*
	VerifySetNftPriceTx:
	accounts order is:
	- FromAccount
		- Assets:
			- AssetGas
		- Nft
			- nft index
	- GasAccount
		- Assets:
			- AssetGas
*/
func VerifySetNftPriceTxInfo(
	accountInfoMap map[int64]*account.Account,
	assetInfoMap map[int64]map[int64]*asset.AccountAsset,
	nftInfoMap map[int64]*nft.L2Nft,
	txInfo *SetNftPriceTxInfo,
) (txDetails []*MempoolTxDetail, err error) {
	// verify params
	if accountInfoMap[txInfo.AccountIndex] == nil ||
		accountInfoMap[txInfo.GasAccountIndex] == nil ||
		assetInfoMap[txInfo.AccountIndex] == nil ||
		assetInfoMap[txInfo.AccountIndex][txInfo.GasFeeAssetId] == nil ||
		assetInfoMap[txInfo.GasAccountIndex] == nil ||
		assetInfoMap[txInfo.GasAccountIndex][txInfo.GasFeeAssetId] == nil ||
		nftInfoMap[txInfo.AccountIndex] == nil ||
		nftInfoMap[txInfo.NftIndex].OwnerAccountIndex != txInfo.AccountIndex ||
		nftInfoMap[txInfo.NftIndex].NftIndex != txInfo.NftIndex ||
		nftInfoMap[txInfo.NftIndex].NftContentHash != txInfo.NftContentHash ||
		txInfo.GasFeeAssetAmount.Cmp(ZeroBigInt) < 0 ||
		txInfo.AssetAmount.Cmp(ZeroBigInt) < 0 {
		logx.Errorf("[VerifySetNftPriceTxInfo] invalid params")
		return nil, errors.New("[VerifySetNftPriceTxInfo] invalid params")
	}
	// verify nonce
	if txInfo.Nonce != accountInfoMap[txInfo.AccountIndex].Nonce {
		log.Println("[VerifySetNftPriceTxInfo] invalid nonce")
		return nil, errors.New("[VerifySetNftPriceTxInfo] invalid nonce")
	}
	// set tx info
	var (
		assetDeltaMap = make(map[int64]map[int64]*big.Int)
		newNftInfo    *NftInfo
	)
	// init delta map
	assetDeltaMap[txInfo.AccountIndex] = make(map[int64]*big.Int)
	if assetDeltaMap[txInfo.GasAccountIndex] == nil {
		assetDeltaMap[txInfo.GasAccountIndex] = make(map[int64]*big.Int)
	}
	// from account asset Gas
	assetDeltaMap[txInfo.AccountIndex][txInfo.GasFeeAssetId] = ffmath.Neg(txInfo.GasFeeAssetAmount)
	// to account nft info
	newNftInfo = &NftInfo{
		NftIndex:            nftInfoMap[txInfo.NftIndex].NftIndex,
		CreatorAccountIndex: nftInfoMap[txInfo.NftIndex].CreatorAccountIndex,
		OwnerAccountIndex:   nftInfoMap[txInfo.NftIndex].OwnerAccountIndex,
		AssetId:             txInfo.AssetId,
		AssetAmount:         txInfo.AssetAmount.String(),
		NftContentHash:      nftInfoMap[txInfo.NftIndex].NftContentHash,
		NftL1TokenId:        nftInfoMap[txInfo.NftIndex].NftL1TokenId,
		NftL1Address:        nftInfoMap[txInfo.NftIndex].NftL1Address,
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
	assetGasBalance, isValid := new(big.Int).SetString(assetInfoMap[txInfo.AccountIndex][txInfo.GasFeeAssetId].Balance, 10)
	if !isValid {
		logx.Errorf("[VerifyMintNftTxInfo] unable to parse balance")
		return nil, errors.New("[VerifyMintNftTxInfo] unable to parse balance")
	}
	if assetGasBalance.Cmp(txInfo.GasFeeAssetAmount) < 0 {
		logx.Errorf("[VerifyMintNftTxInfo] you don't have enough balance of asset Gas")
		return nil, errors.New("[VerifyMintNftTxInfo] you don't have enough balance of asset Gas")
	}
	// compute hash
	hFunc := mimc.NewMiMC()
	msgHash := legendTxTypes.ComputeSetNftPriceMsgHash(txInfo, hFunc)
	// verify signature
	hFunc.Reset()
	pk, err := ParsePkStr(accountInfoMap[txInfo.AccountIndex].PublicKey)
	if err != nil {
		return nil, err
	}
	isValid, err = pk.Verify(txInfo.Sig, msgHash, hFunc)
	if err != nil {
		log.Println("[VerifySetNftPriceTxInfo] unable to verify signature:", err)
		return nil, err
	}
	if !isValid {
		log.Println("[VerifySetNftPriceTxInfo] invalid signature")
		return nil, errors.New("[VerifySetNftPriceTxInfo] invalid signature")
	}
	// compute tx details
	// from account asset gas
	txDetails = append(txDetails, &MempoolTxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    GeneralAssetType,
		AccountIndex: txInfo.AccountIndex,
		AccountName:  accountInfoMap[txInfo.AccountIndex].AccountName,
		Balance:      assetInfoMap[txInfo.AccountIndex][txInfo.GasFeeAssetId].Balance,
		BalanceDelta: assetDeltaMap[txInfo.AccountIndex][txInfo.GasFeeAssetId].String(),
	})
	// from account nft delta
	oldNftInfo := &NftInfo{
		NftIndex:            nftInfoMap[txInfo.NftIndex].NftIndex,
		CreatorAccountIndex: nftInfoMap[txInfo.NftIndex].CreatorAccountIndex,
		OwnerAccountIndex:   nftInfoMap[txInfo.NftIndex].OwnerAccountIndex,
		AssetId:             nftInfoMap[txInfo.NftIndex].AssetId,
		AssetAmount:         nftInfoMap[txInfo.NftIndex].AssetAmount,
		NftContentHash:      nftInfoMap[txInfo.NftIndex].NftContentHash,
		NftL1TokenId:        nftInfoMap[txInfo.NftIndex].NftL1TokenId,
		NftL1Address:        nftInfoMap[txInfo.NftIndex].NftL1Address,
	}
	txDetails = append(txDetails, &MempoolTxDetail{
		AssetId:      txInfo.NftIndex,
		AssetType:    NftAssetType,
		AccountIndex: txInfo.AccountIndex,
		AccountName:  accountInfoMap[txInfo.AccountIndex].AccountName,
		Balance:      oldNftInfo.String(),
		BalanceDelta: newNftInfo.String(),
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
