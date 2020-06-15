package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/docker/api/azure"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/resources"
	azure_storage "github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/storage/mgmt/storage"
	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/docker/api/context/store"
	"github.com/docker/api/tests/aci-e2e/storage"
	. "github.com/docker/api/tests/framework"
)

const (
	resourceGroupName = "resourceGroupTest"
	location          = "westeurope"
	contextName       = "acitest"

	testContainerName = "testcontainername"
)

var (
	subscriptionID string
)

type E2eACISuite struct {
	Suite
}

func (s *E2eACISuite) TestContextDefault() {
	It("should be initialized with default context", func() {
		_, err := s.NewCommand("docker", "context", "rm", "-f", contextName).Exec()
		if err == nil {
			log.Println("Cleaning existing test context")
		}

		s.NewCommand("docker", "context", "use", "default").ExecOrDie()
		output := s.NewCommand("docker", "context", "ls").ExecOrDie()
		Expect(output).To(Not(ContainSubstring(contextName)))
		Expect(output).To(ContainSubstring("default *"))
	})
}

func (s *E2eACISuite) TestACIBackend() {
	It("creates a new aci context for tests", func() {
		setupTestResourceGroup(resourceGroupName)
		helper := azure.NewACIResourceGroupHelper()
		models, err := helper.GetSubscriptionIDs(context.TODO())
		Expect(err).To(BeNil())
		subscriptionID = *models[0].SubscriptionID

		s.NewDockerCommand("context", "create", "aci", contextName, "--subscription-id", subscriptionID, "--resource-group", resourceGroupName, "--location", location).ExecOrDie()
		// Expect(output).To(ContainSubstring("ACI context acitest created"))
	})

	defer deleteResourceGroup(resourceGroupName)

	It("uses the aci context", func() {
		currentContext := s.NewCommand("docker", "context", "use", contextName).ExecOrDie()
		Expect(currentContext).To(ContainSubstring(contextName))
		output := s.NewCommand("docker", "context", "ls").ExecOrDie()
		Expect(output).To(ContainSubstring("acitest *"))
	})

	It("ensures no container is running initially", func() {
		output := s.NewDockerCommand("ps").ExecOrDie()
		Expect(len(Lines(output))).To(Equal(1))
	})

	It("runs nginx on port 80", func() {
		aciContext := store.AciContext{
			SubscriptionID: subscriptionID,
			Location:       location,
			ResourceGroup:  resourceGroupName,
		}
		createStorageAccount(aciContext, testStorageAccountName)
		defer deleteStorageAccount(aciContext)
		keys := getStorageKeys(aciContext, testStorageAccountName)
		firstKey := *keys[0].Value
		credential, u := createFileShare(firstKey, testShareName)
		uploadFile(credential, u.String(), testFileName, testFileContent)

		mountTarget := "/usr/share/nginx/html"
		output := s.NewDockerCommand("run", "nginx",
			"-v", fmt.Sprintf("%s:%s@%s:%s",
				testStorageAccountName, firstKey, testShareName, mountTarget),
			"-p", "80:80",
			"--name", testContainerName).ExecOrDie()
		Expect(output).To(Equal(testContainerName + "\n"))
		output = s.NewDockerCommand("ps").ExecOrDie()
		lines := Lines(output)
		Expect(len(lines)).To(Equal(2))

		containerFields := Columns(lines[1])
		Expect(containerFields[1]).To(Equal("nginx"))
		Expect(containerFields[2]).To(Equal("Running"))
		exposedIP := containerFields[3]
		containerID := containerFields[0]
		Expect(exposedIP).To(ContainSubstring(":80->80/tcp"))

		publishedURL := strings.ReplaceAll(exposedIP, "->80/tcp", "")
		output = s.NewCommand("curl", publishedURL).ExecOrDie()
		Expect(output).To(ContainSubstring(testFileContent))

		output = s.NewDockerCommand("logs", containerID).ExecOrDie()
		Expect(output).To(ContainSubstring("GET"))
	})

	It("removes container nginx", func() {
		output := s.NewDockerCommand("rm", testContainerName).ExecOrDie()
		Expect(Lines(output)[0]).To(Equal(testContainerName))
	})

	It("deploys a compose app", func() {
		s.NewDockerCommand("compose", "up", "-f", "../composefiles/aci-demo/aci_demo_port.yaml", "--name", "acidemo").ExecOrDie()
		// Expect(output).To(ContainSubstring("Successfully deployed"))
		output := s.NewDockerCommand("ps").ExecOrDie()
		Lines := Lines(output)
		Expect(len(Lines)).To(Equal(4))
		webChecked := false

		for _, line := range Lines[1:] {
			Expect(line).To(ContainSubstring("Running"))
			if strings.Contains(line, "acidemo_web") {
				webChecked = true
				containerFields := Columns(line)
				exposedIP := containerFields[3]
				Expect(exposedIP).To(ContainSubstring(":80->80/tcp"))

				url := strings.ReplaceAll(exposedIP, "->80/tcp", "")
				output = s.NewCommand("curl", url).ExecOrDie()
				Expect(output).To(ContainSubstring("Docker Compose demo"))
				output = s.NewCommand("curl", url+"/words/noun").ExecOrDie()
				Expect(output).To(ContainSubstring("\"word\":"))
			}
		}

		Expect(webChecked).To(BeTrue())
	})

	It("get logs from web service", func() {
		output := s.NewDockerCommand("logs", "acidemo_web").ExecOrDie()
		Expect(output).To(ContainSubstring("Listening on port 80"))
	})

	It("shutdown compose app", func() {
		s.NewDockerCommand("compose", "down", "-f", "../composefiles/aci-demo/aci_demo_port.yaml", "--name", "acidemo").ExecOrDie()
	})
	It("switches back to default context", func() {
		output := s.NewCommand("docker", "context", "use", "default").ExecOrDie()
		Expect(output).To(ContainSubstring("default"))
	})

	It("deletes test context", func() {
		output := s.NewCommand("docker", "context", "rm", contextName).ExecOrDie()
		Expect(output).To(ContainSubstring(contextName))
	})
}

const (
	testStorageAccountName = "dockertestaccount"
	testShareName          = "dockertestshare"
	testFileContent        = "Volume mounted with success!"
	testFileName           = "index.html"
)

func createStorageAccount(aciContext store.AciContext, accountName string) azure_storage.Account {
	log.Println("Creating storage account " + accountName)
	storageAccount, err := storage.CreateStorageAccount(context.TODO(), aciContext, accountName)
	Expect(err).To(BeNil())
	Expect(*storageAccount.Name).To(Equal(accountName))
	return storageAccount
}

func getStorageKeys(aciContext store.AciContext, storageAccountName string) []azure_storage.AccountKey {
	list, err := storage.ListKeys(context.TODO(), aciContext, storageAccountName)
	Expect(err).To(BeNil())
	Expect(list.Keys).ToNot(BeNil())
	Expect(len(*list.Keys)).To(BeNumerically(">", 0))

	return *list.Keys
}

func deleteStorageAccount(aciContext store.AciContext) {
	log.Println("Deleting storage account " + testStorageAccountName)
	_, err := storage.DeleteStorageAccount(context.TODO(), aciContext, testStorageAccountName)
	Expect(err).To(BeNil())
}

func createFileShare(key, shareName string) (azfile.SharedKeyCredential, url.URL) {
	// Create a ShareURL object that wraps a soon-to-be-created share's URL and a default pipeline.
	u, _ := url.Parse(fmt.Sprintf("https://%s.file.core.windows.net/%s", testStorageAccountName, shareName))
	credential, err := azfile.NewSharedKeyCredential(testStorageAccountName, key)
	Expect(err).To(BeNil())

	shareURL := azfile.NewShareURL(*u, azfile.NewPipeline(credential, azfile.PipelineOptions{}))
	_, err = shareURL.Create(context.TODO(), azfile.Metadata{}, 0)
	Expect(err).To(BeNil())

	return *credential, *u
}

func uploadFile(credential azfile.SharedKeyCredential, baseURL, fileName, fileContent string) {
	fURL, err := url.Parse(baseURL + "/" + fileName)
	Expect(err).To(BeNil())
	fileURL := azfile.NewFileURL(*fURL, azfile.NewPipeline(&credential, azfile.PipelineOptions{}))
	err = azfile.UploadBufferToAzureFile(context.TODO(), []byte(fileContent), fileURL, azfile.UploadToAzureFileOptions{})
	Expect(err).To(BeNil())
}

func TestE2eACI(t *testing.T) {
	suite.Run(t, new(E2eACISuite))
}

func setupTestResourceGroup(groupName string) {
	log.Println("Creating resource group " + resourceGroupName)
	ctx := context.TODO()
	helper := azure.NewACIResourceGroupHelper()
	models, err := helper.GetSubscriptionIDs(ctx)
	Expect(err).To(BeNil())
	_, err = helper.CreateOrUpdate(ctx, *models[0].SubscriptionID, groupName, resources.Group{
		Location: to.StringPtr(location),
	})
	Expect(err).To(BeNil())
}

func deleteResourceGroup(groupName string) {
	log.Println("Deleting resource group " + resourceGroupName)
	ctx := context.TODO()
	helper := azure.NewACIResourceGroupHelper()
	models, err := helper.GetSubscriptionIDs(ctx)
	Expect(err).To(BeNil())
	err = helper.Delete(ctx, *models[0].SubscriptionID, groupName)
	Expect(err).To(BeNil())
}
