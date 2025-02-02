package types

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var errorSinglSignerExpected = errors.New("only a single signer is supported")

// VerifySig checks that the signature over the provided transaction is valid using the provided signer data.
func VerifySig(signerData authsigning.SignerData, txConfig client.TxConfig, authTx authsigning.Tx) (bool, error) {
	signBytes, err := txConfig.SignModeHandler().GetSignBytes(signing.SignMode_SIGN_MODE_DIRECT, signerData, authTx)
	if err != nil {
		return false, err
	}

	sigs, err := authTx.GetSignaturesV2()
	if err != nil {
		return false, err
	}
	if len(sigs) != 1 {
		return false, errorSinglSignerExpected
	}

	sigData := sigs[0].Data

	rawSig, ok := sigData.(*signing.SingleSignatureData)
	if !ok {
		return false, errorSinglSignerExpected
	}

	return signerData.PubKey.VerifySignature(signBytes, rawSig.Signature), nil
}

// VerifyPFBSigs checks that all of the signatures for a transaction that
// contains a MsgWirePayForBlob message by going through the entire malleation
// process.
func VerifyPFBSigs(signerData authsigning.SignerData, txConfig client.TxConfig, wirePFBTx authsigning.Tx) (bool, error) {
	wirePFBMsg, err := ExtractMsgWirePayForBlob(wirePFBTx)
	if err != nil {
		return false, err
	}

	// go through the entire malleation process as if this tx was being included in a block.
	_, pfb, sig, err := ProcessWireMsgPayForBlob(wirePFBMsg)
	if err != nil {
		return false, err
	}

	// create the malleated MsgPayForBlob tx by using auth data from the original tx
	pfbTx, err := BuildPayForBlobTxFromWireTx(wirePFBTx, txConfig.NewTxBuilder(), sig, pfb)
	if err != nil {
		return false, err
	}

	valid, err := VerifySig(signerData, txConfig, pfbTx)
	if err != nil || !valid {
		return false, err
	}

	return true, nil
}
