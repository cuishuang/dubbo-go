/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package remote

import (
	"github.com/apache/dubbo-go/registry"
	"sync"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/metadata/definition"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report/delegate"
	"github.com/apache/dubbo-go/metadata/service/inmemory"
)

// version will be used by Version func
const (
	version = "1.0.0"
	remote  = "remote"
)

// MetadataService is a implement of metadata service which will delegate the remote metadata report
// This is singleton
type RemoteMetadataService struct {
	inMemoryMetadataService *inmemory.MetadataService
	exportedRevision        atomic.String
	subscribedRevision      atomic.String
	delegateReport          *delegate.MetadataReport
}

var (
	metadataServiceOnce     sync.Once
	metadataServiceInstance *RemoteMetadataService
)

// GetRemoteMetadataService will create a new remote MetadataService instance
func GetRemoteMetadataService() (*RemoteMetadataService, error) {
	var err error
	metadataServiceOnce.Do(func() {
		var mr *delegate.MetadataReport
		mr, err = delegate.NewMetadataReport()
		if err != nil {
			return
		}
		// it will never return error
		inms, _ := inmemory.GetInMemoryMetadataService()
		metadataServiceInstance = &RemoteMetadataService{
			inMemoryMetadataService: inms.(*inmemory.MetadataService),
			delegateReport:          mr,
		}
	})
	return metadataServiceInstance, err
}

// publishMetadata
func (mts *RemoteMetadataService) PublishMetadata(service string) {
	info := mts.inMemoryMetadataService.GetMetadataInfo("")
	if info.HasReported() {
		return
	}
	id := identifier.NewSubscriberMetadataIdentifier(service, info.CalAndGetRevision())
	err := mts.delegateReport.PublishAppMetadata(id, info)
	if err != nil {
		logger.Errorf("Publishing metadata to error[%v]", err)
		return
	}
	info.MarkReported()
}

// publishMetadata
func (mts *RemoteMetadataService) GetMetadata(instance registry.ServiceInstance) (*common.MetadataInfo, error) {
	revision := instance.GetMetadata()[constant.EXPORTED_SERVICES_REVISION_PROPERTY_NAME]
	id := identifier.NewSubscriberMetadataIdentifier(instance.GetServiceName(), revision)
	return mts.delegateReport.GetAppMetadata(id)
}

// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
func (mts *RemoteMetadataService) PublishServiceDefinition(url *common.URL) error {
	interfaceName := url.GetParam(constant.INTERFACE_KEY, "")
	isGeneric := url.GetParamBool(constant.GENERIC_KEY, false)
	if common.RoleType(common.PROVIDER).Role() == url.GetParam(constant.SIDE_KEY, "") {
		if len(interfaceName) > 0 && !isGeneric {
			sv := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
			sd := definition.BuildServiceDefinition(*sv, url)
			id := &identifier.MetadataIdentifier{
				BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
					ServiceInterface: interfaceName,
					Version:          url.GetParam(constant.VERSION_KEY, ""),
					Group:            url.GetParam(constant.GROUP_KEY, constant.DUBBO),
					Side:             url.GetParam(constant.SIDE_KEY, constant.PROVIDER_PROTOCOL),
				},
			}
			mts.delegateReport.StoreProviderMetadata(id, sd)
			return nil
		}
		logger.Errorf("publishProvider interfaceName is empty . providerUrl:%v ", url)
	} else {
		params := make(map[string]string, len(url.GetParams()))
		url.RangeParams(func(key, value string) bool {
			params[key] = value
			return true
		})
		id := &identifier.MetadataIdentifier{
			BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
				ServiceInterface: interfaceName,
				Version:          url.GetParam(constant.VERSION_KEY, ""),
				Group:            url.GetParam(constant.GROUP_KEY, constant.DUBBO),
				Side:             url.GetParam(constant.SIDE_KEY, "consumer"),
			},
		}
		mts.delegateReport.StoreConsumerMetadata(id, params)
		return nil
	}

	return nil
}

// GetExportedURLs will be implemented by in memory service
func (mts *RemoteMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error) {
	return mts.inMemoryMetadataService.GetExportedURLs(serviceInterface, group, version, protocol)
}

// GetSubscribedURLs will be implemented by in memory service
func (mts *RemoteMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	return mts.inMemoryMetadataService.GetSubscribedURLs()
}

// GetServiceDefinition will be implemented by in memory service
func (mts *RemoteMetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	return mts.inMemoryMetadataService.GetServiceDefinition(interfaceName, group, version)
}

// GetServiceDefinitionByServiceKey will be implemented by in memory service
func (mts *RemoteMetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	return mts.inMemoryMetadataService.GetServiceDefinitionByServiceKey(serviceKey)
}
