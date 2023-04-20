// Copyright © Microsoft <wastore@microsoft.com>
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

package e2etest

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	blobsas "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	blobservice "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-storage-azcopy/v10/azbfs"
	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/google/uuid"
)

// provide convenient methods to get access to test resources such as accounts, containers/shares, directories
type TestResourceFactory struct{}

func (TestResourceFactory) GetBlobServiceURL(accountType AccountType) *blobservice.Client {
	accountName, accountKey := GlobalInputManager{}.GetAccountAndKey(accountType)
	resourceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	credential, err := blob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		panic(err)
	}
	bsc, err := blobservice.NewClientWithSharedKeyCredential(resourceURL, credential, nil)
	if err != nil {
		panic(err)
	}
	return bsc
}

func (TestResourceFactory) GetFileServiceURL(accountType AccountType) azfile.ServiceURL {
	accountName, accountKey := GlobalInputManager{}.GetAccountAndKey(accountType)
	u, _ := url.Parse(fmt.Sprintf("https://%s.file.core.windows.net/", accountName))

	credential, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		panic(err)
	}
	pipeline := azfile.NewPipeline(credential, azfile.PipelineOptions{})
	return azfile.NewServiceURL(*u, pipeline)
}

func (TestResourceFactory) GetDatalakeServiceURL(accountType AccountType) azbfs.ServiceURL {
	accountName, accountKey := GlobalInputManager{}.GetAccountAndKey(accountType)
	u, _ := url.Parse(fmt.Sprintf("https://%s.dfs.core.windows.net/", accountName))

	cred := azbfs.NewSharedKeyCredential(accountName, accountKey)
	pipeline := azbfs.NewPipeline(cred, azbfs.PipelineOptions{})
	return azbfs.NewServiceURL(*u, pipeline)
}

func (TestResourceFactory) GetBlobServiceURLWithSAS(c asserter, accountType AccountType) *blobservice.Client {
	accountName, accountKey := GlobalInputManager{}.GetAccountAndKey(accountType)
	credential, err := blob.NewSharedKeyCredential(accountName, accountKey)
	c.AssertNoErr(err)
	rawURL := fmt.Sprintf("https://%s.blob.core.windows.net/", credential.AccountName())
	client, err := blobservice.NewClientWithSharedKeyCredential(rawURL, credential, nil)
	c.AssertNoErr(err)

	sasURL, err := client.GetSASURL(
		blobsas.AccountResourceTypes{Service: true, Container: true, Object: true},
		blobsas.AccountPermissions{Read: true, List: true, Write: true, Delete: true, DeletePreviousVersion: true, Add: true, Create: true, Update: true, Process: true, Tag: true, FilterByTags: true},
		time.Now().Add(48 * time.Hour),
		nil)
	c.AssertNoErr(err)

	client, err = blobservice.NewClientWithNoCredential(sasURL, nil)
	c.AssertNoErr(err)

	return client
}

func (TestResourceFactory) GetContainerURLWithSAS(c asserter, accountType AccountType, containerName string) *container.Client {
	accountName, accountKey := GlobalInputManager{}.GetAccountAndKey(accountType)
	credential, err := blob.NewSharedKeyCredential(accountName, accountKey)
	c.AssertNoErr(err)
	rawURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", credential.AccountName(), containerName)
	client, err := container.NewClientWithSharedKeyCredential(rawURL, credential, nil)
	c.AssertNoErr(err)

	sasURL, err := client.GetSASURL(
		blobsas.ContainerPermissions{Read: true, Add: true, Write: true, Create: true, Delete: true, DeletePreviousVersion: true, List: true, ModifyOwnership: true, ModifyPermissions: true},
		time.Now().Add(48 * time.Hour),
		nil)
	c.AssertNoErr(err)

	client, err = container.NewClientWithNoCredential(sasURL, nil)
	c.AssertNoErr(err)

	return client
}

func (TestResourceFactory) GetFileShareULWithSAS(c asserter, accountType AccountType, containerName string) azfile.ShareURL {
	accountName, accountKey := GlobalInputManager{}.GetAccountAndKey(accountType)
	credential, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	c.AssertNoErr(err)

	sasQueryParams, err := azfile.FileSASSignatureValues{
		Protocol:    azfile.SASProtocolHTTPS,
		ExpiryTime:  time.Now().UTC().Add(48 * time.Hour),
		ShareName:   containerName,
		Permissions: azfile.ShareSASPermissions{Read: true, Write: true, Create: true, Delete: true, List: true}.String(),
	}.NewSASQueryParameters(credential)
	c.AssertNoErr(err)

	// construct the url from scratch
	qp := sasQueryParams.Encode()
	rawURL := fmt.Sprintf("https://%s.file.core.windows.net/%s?%s",
		credential.AccountName(), containerName, qp)

	// convert the raw url and validate it was parsed successfully
	fullURL, err := url.Parse(rawURL)
	c.AssertNoErr(err)

	return azfile.NewShareURL(*fullURL, azfile.NewPipeline(azfile.NewAnonymousCredential(), azfile.PipelineOptions{}))
}

func (TestResourceFactory) GetBlobURLWithSAS(c asserter, accountType AccountType, containerName string, blobName string) *blob.Client {
	containerURLWithSAS := TestResourceFactory{}.GetContainerURLWithSAS(c, accountType, containerName)
	blobURLWithSAS := containerURLWithSAS.NewBlobClient(blobName)
	return blobURLWithSAS
}

func (TestResourceFactory) CreateNewContainer(c asserter, publicAccess container.PublicAccessType, accountType AccountType) (cc *container.Client, name string, rawURL string) {
	name = TestResourceNameGenerator{}.GenerateContainerName(c)
	cc = TestResourceFactory{}.GetBlobServiceURL(accountType).NewContainerClient(name)

	_, err := cc.Create(context.Background(), &container.CreateOptions{Access: &publicAccess})
	c.AssertNoErr(err)
	return cc, name, TestResourceFactory{}.GetContainerURLWithSAS(c, accountType, name).URL()
}

const defaultShareQuotaGB = 512

func (TestResourceFactory) CreateNewFileShare(c asserter, accountType AccountType) (fileShare azfile.ShareURL, name string, rawSasURL url.URL) {
	name = TestResourceNameGenerator{}.GenerateContainerName(c)
	fileShare = TestResourceFactory{}.GetFileServiceURL(accountType).NewShareURL(name)

	cResp, err := fileShare.Create(context.Background(), nil, defaultShareQuotaGB)
	c.AssertNoErr(err)
	c.Assert(cResp.StatusCode(), equals(), 201)
	return fileShare, name, TestResourceFactory{}.GetFileShareULWithSAS(c, accountType, name).URL()
}

func (TestResourceFactory) CreateNewFileShareSnapshot(c asserter, fileShare azfile.ShareURL) (snapshotID string) {
	resp, err := fileShare.CreateSnapshot(context.TODO(), azfile.Metadata{})
	c.AssertNoErr(err)
	return resp.Snapshot()
}

func (TestResourceFactory) CreateLocalDirectory(c asserter) (dstDirName string) {
	dstDirName, err := os.MkdirTemp("", "AzCopyLocalTest")
	c.AssertNoErr(err)
	return
}

type TestResourceNameGenerator struct{}

const (
	containerPrefix = "e2e"
	blobPrefix      = "blob"
)

func getTestName(t *testing.T) (pseudoSuite, test string) {

	removeUnderscores := func(s string) string {
		return strings.Replace(s, "_", "-", -1) // necessary if using name as basis for blob container name
	}

	testName := t.Name()

	// Look up the stack to find out more info about the test method
	// Note: the way to do this changed in go 1.12, refer to release notes for more info
	var pcs [10]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	fileName := ""
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Func.Name(), "."+testName) {
			fileName = frame.File
			break
		} else if !more {
			break
		}
	}

	// When using the basic Testing package, we have adopted a convention that
	// the test name should being with one of the words in the file name, followed by a _ .
	// Try to extract a "pseudo suite" name from the test name according to that rule.
	pseudoSuite = ""
	testName = strings.Replace(testName, "Test", "", 1)
	uscorePos := strings.Index(testName, "_")
	if uscorePos >= 0 && uscorePos < len(testName)-1 {
		beforeUnderscore := strings.ToLower(testName[:uscorePos])

		fileWords := strings.ReplaceAll(
			strings.TrimSuffix(strings.TrimPrefix(path.Base(fileName), "zt_"), "_test.go"), "_", "")

		if strings.Contains(fileWords, beforeUnderscore) {
			pseudoSuite = beforeUnderscore
			testName = testName[uscorePos+1:]
		}
		// fileWords := strings.Split(strings.Replace(strings.ToLower(filepath.Base(fileName)), "_test.go", "", -1), "_")
		// for _, w := range fileWords {
		// 	if beforeUnderscore == w {
		// 		pseudoSuite = beforeUnderscore
		// 		testName = testName[uscorePos+1:]
		// 		break
		// 	}
		// }
	}

	return pseudoSuite, removeUnderscores(testName)
}

// This function generates an entity name by concatenating the passed prefix,
// the name of the test requesting the entity name, and the minute, second, and nanoseconds of the call.
// This should make it easy to associate the entities with their test, uniquely identify
// them, and determine the order in which they were created.
// Will truncate the end of the test name, if there is not enough room for it, followed by the time-based suffix,
// with a non-zero maxLen.
func generateName(c asserter, prefix string, maxLen int) string {
	name := c.CompactScenarioName() // don't want to just use test name here, because each test contains multiple scenarios with the declarative runner

	textualPortion := fmt.Sprintf("%s-%s", prefix, strings.ToLower(name))
	// GUIDs are less prone to overlap than times.
	guidSuffix := uuid.New().String()
	if maxLen > 0 {
		maxTextLen := maxLen - len(guidSuffix)
		if maxTextLen < 1 {
			panic("max len too short")
		}
		if len(textualPortion) > maxTextLen {
			textualPortion = textualPortion[:maxTextLen]
		}
	}
	name = textualPortion + guidSuffix
	return name
}

func (TestResourceNameGenerator) GenerateContainerName(c asserter) string {
	// return generateName(c, containerPrefix, 63)
	return uuid.New().String()
}

func (TestResourceNameGenerator) generateBlobName(c asserter) string {
	return generateName(c, blobPrefix, 0)
}
