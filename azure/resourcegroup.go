/*
   Copyright 2020 Docker, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/subscription/mgmt/subscription"
	"github.com/pkg/errors"

	"github.com/docker/api/errdefs"
)

// ACIResourceGroupHelper interface to manage resource groups and subscription IDs
type ACIResourceGroupHelper interface {
	GetSubscriptionIDs(ctx context.Context) ([]subscription.Model, error)
	ListGroups(ctx context.Context, subscriptionID string) ([]resources.Group, error)
	GetGroup(ctx context.Context, subscriptionID string, groupName string) (resources.Group, error)
	CreateOrUpdate(ctx context.Context, subscriptionID string, resourceGroupName string, parameters resources.Group) (result resources.Group, err error)
	Delete(ctx context.Context, subscriptionID string, resourceGroupName string) error
}

type aciResourceGroupHelperImpl struct {
}

// NewACIResourceGroupHelper create a new ACIResourceGroupHelper
func NewACIResourceGroupHelper() ACIResourceGroupHelper {
	return aciResourceGroupHelperImpl{}
}

// GetGroup get a resource group from its name
func (mgt aciResourceGroupHelperImpl) GetGroup(ctx context.Context, subscriptionID string, groupName string) (resources.Group, error) {
	gc, err := getGroupsClient(subscriptionID)
	if err != nil {
		return resources.Group{}, err
	}
	return gc.Get(ctx, groupName)
}

// ListGroups list resource groups
func (mgt aciResourceGroupHelperImpl) ListGroups(ctx context.Context, subscriptionID string) ([]resources.Group, error) {
	gc, err := getGroupsClient(subscriptionID)
	if err != nil {
		return nil, err
	}

	groupResponse, err := gc.List(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	groups := groupResponse.Values()

	for groupResponse.NotDone() {
		err = groupResponse.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
		newValues := groupResponse.Values()
		groups = append(groups, newValues...)
	}

	return groups, nil
}

// CreateOrUpdate create or update a resource group
func (mgt aciResourceGroupHelperImpl) CreateOrUpdate(ctx context.Context, subscriptionID string, resourceGroupName string, parameters resources.Group) (result resources.Group, err error) {
	gc, err := getGroupsClient(subscriptionID)
	if err != nil {
		return resources.Group{}, err
	}
	return gc.CreateOrUpdate(ctx, resourceGroupName, parameters)
}

// Delete deletes a resource group
func (mgt aciResourceGroupHelperImpl) Delete(ctx context.Context, subscriptionID string, resourceGroupName string) (err error) {
	gc, err := getGroupsClient(subscriptionID)
	if err != nil {
		return err
	}

	future, err := gc.Delete(ctx, resourceGroupName)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, gc.Client)
}

// GetSubscriptionIDs Return available subscription IDs based on azure login
func (mgt aciResourceGroupHelperImpl) GetSubscriptionIDs(ctx context.Context) ([]subscription.Model, error) {
	c, err := getSubscriptionsClient()
	if err != nil {
		return nil, err
	}
	res, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	subs := res.Values()

	if len(subs) == 0 {
		return nil, errors.New("no subscriptions found")
	}
	for res.NotDone() {
		err = res.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
		subs = append(subs, res.Values()...)
	}
	return subs, nil
}

func getSubscriptionsClient() (subscription.SubscriptionsClient, error) {
	subc := subscription.NewSubscriptionsClient()
	err := setupClient(&subc.Client)
	if err != nil {
		return subscription.SubscriptionsClient{}, errors.Wrap(errdefs.ErrLoginFailed, err.Error())
	}
	return subc, nil
}

func getGroupsClient(subscriptionID string) (resources.GroupsClient, error) {
	groupsClient := resources.NewGroupsClient(subscriptionID)
	err := setupClient(&groupsClient.Client)
	if err != nil {
		return resources.GroupsClient{}, err
	}
	return groupsClient, nil
}
