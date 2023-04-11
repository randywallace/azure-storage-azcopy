// Copyright © 2017 Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package common

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	blobservice "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

var glcm = GetLifecycleMgr()

// newClientCallbacks is a Generic Type to allow client creation error handling to live in a single place (createClient)
// T = Client type
// U = SharedKeyCredential type
// Note : Could also make azcore.ClientOptions generic here if one day different storage service clients have additional options. This would also make the callback definitions easier.
type newClientCallbacks[T, U any] struct {
	TokenCredential        func(string, azcore.TokenCredential, azcore.ClientOptions) (*T, error)
	NoCredential           func(string, azcore.ClientOptions) (*T, error)
}

// createClient is a generic method to allow client creation error handling to live in a single place
func createClient[T, U any](callbacks newClientCallbacks[T, U], u string, credInfo CredentialInfo, credOpOptions *CredentialOpOptions, options azcore.ClientOptions) (*T, error) {
	if credOpOptions == nil {
		credOpOptions = &CredentialOpOptions{
			LogError: glcm.Info,
		}
	}
	switch credInfo.CredentialType {
	case ECredentialType.OAuthToken():
		if credInfo.OAuthTokenInfo.IsEmpty() {
			credOpOptions.panicError(errors.New("invalid state, cannot get valid OAuth token information"))
		}
		tc, err := credInfo.OAuthTokenInfo.GetTokenCredential()
		if err != nil {
			credOpOptions.panicError(fmt.Errorf("unable to get token credential due to reason (%s)", err.Error()))
		}
		return callbacks.TokenCredential(u, tc, options)
	case ECredentialType.Anonymous():
		return callbacks.NoCredential(u, options)
	default:
		credOpOptions.panicError(fmt.Errorf("invalid state, credential type %v is not supported", credInfo.CredentialType))
		return nil, fmt.Errorf("invalid state, credential type %v is not supported", credInfo.CredentialType)
	}
}

///////////////////////////////////////////////// BLOB FUNCTIONS /////////////////////////////////////////////////

// CreateBlobServiceClient creates a blob service client with credentials specified by credInfo
func CreateBlobServiceClient(u string, credInfo CredentialInfo, credOpOptions *CredentialOpOptions, options azcore.ClientOptions) (*blobservice.Client, error) {
	callbacks := newClientCallbacks[blobservice.Client, blob.SharedKeyCredential]{
		TokenCredential: func(u string, tc azcore.TokenCredential, options azcore.ClientOptions) (*blobservice.Client, error) {
			return blobservice.NewClient(u, tc, &blobservice.ClientOptions{ClientOptions: options})
		},
		NoCredential: func(u string, options azcore.ClientOptions) (*blobservice.Client, error) {
			return blobservice.NewClientWithNoCredential(u, &blobservice.ClientOptions{ClientOptions: options})
		},
	}

	return createClient(callbacks, u, credInfo, credOpOptions, options)
}

func CreateContainerClient(u string, credInfo CredentialInfo, credOpOptions *CredentialOpOptions, options azcore.ClientOptions) (*container.Client, error) {
	callbacks := newClientCallbacks[container.Client, blob.SharedKeyCredential]{
		TokenCredential: func(u string, tc azcore.TokenCredential, options azcore.ClientOptions) (*container.Client, error) {
			return container.NewClient(u, tc, &container.ClientOptions{ClientOptions: options})
		},
		NoCredential: func(u string, options azcore.ClientOptions) (*container.Client, error) {
			return container.NewClientWithNoCredential(u, &container.ClientOptions{ClientOptions: options})
		},
	}

	return createClient(callbacks, u, credInfo, credOpOptions, options)
}

func CreateBlobClient(u string, credInfo CredentialInfo, credOpOptions *CredentialOpOptions, options azcore.ClientOptions) (*blob.Client, error) {
	callbacks := newClientCallbacks[blob.Client, blob.SharedKeyCredential]{
		TokenCredential: func(u string, tc azcore.TokenCredential, options azcore.ClientOptions) (*blob.Client, error) {
			return blob.NewClient(u, tc, &blob.ClientOptions{ClientOptions: options})
		},
		NoCredential: func(u string, options azcore.ClientOptions) (*blob.Client, error) {
			return blob.NewClientWithNoCredential(u, &blob.ClientOptions{ClientOptions: options})
		},
	}

	return createClient(callbacks, u, credInfo, credOpOptions, options)
}

func CreateBlockBlobClient(u string, credInfo CredentialInfo, credOpOptions *CredentialOpOptions, options azcore.ClientOptions) (*blockblob.Client, error) {
	callbacks := newClientCallbacks[blockblob.Client, blob.SharedKeyCredential]{
		TokenCredential: func(u string, tc azcore.TokenCredential, options azcore.ClientOptions) (*blockblob.Client, error) {
			return blockblob.NewClient(u, tc, &blockblob.ClientOptions{ClientOptions: options})
		},
		NoCredential: func(u string, options azcore.ClientOptions) (*blockblob.Client, error) {
			return blockblob.NewClientWithNoCredential(u, &blockblob.ClientOptions{ClientOptions: options})
		},
	}

	return createClient(callbacks, u, credInfo, credOpOptions, options)
}
