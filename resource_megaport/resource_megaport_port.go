// Copyright 2020 Megaport Pty Ltd
//
// Licensed under the Mozilla Public License, Version 2.0 (the
// "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//       https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource_megaport

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportPort() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportPortCreate,
		Read:   resourceMegaportPortRead,
		Update: resourceMegaportPortUpdate,
		Delete: resourceMegaportPortDelete,
		Schema: schema_megaport.ResourcePortSchema(),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceMegaportPortCreate(d *schema.ResourceData, m interface{}) error {
	var portId string
	var portErr error

	port := m.(*terraform_utility.MegaportClient).Port
	location := m.(*terraform_utility.MegaportClient).Location

	portName := d.Get("port_name").(string)
	term := d.Get("term").(int)
	portSpeed := d.Get("port_speed").(int)
	locationId := d.Get("location_id").(int)
	marketplaceVisibility := d.Get("marketplace_visibility").(bool)
	isLag := d.Get("lag").(bool)
	numberOfPorts := d.Get("lag_port_count").(int)
	loc, locationErr := location.GetLocationByID(locationId)
	diversityZone := d.Get("diversity_zone").(string)

	if locationErr != nil {
		return locationErr
	}

	marketCode := loc.Market

	if isLag {
		if len(diversityZone) > 0 {
			portId, portErr = port.BuyZonedLAGPort(portName, term, portSpeed, locationId, marketCode, numberOfPorts, !marketplaceVisibility, diversityZone)
		} else {
			portId, portErr = port.BuyLAGPort(portName, term, portSpeed, locationId, marketCode, numberOfPorts, !marketplaceVisibility)
		}
	} else {
		if len(diversityZone) > 0 {
			portId, portErr = port.BuyZonedSinglePort(portName, term, portSpeed, locationId, marketCode, !marketplaceVisibility, diversityZone)
		} else {
			portId, portErr = port.BuySinglePort(portName, term, portSpeed, locationId, marketCode, !marketplaceVisibility)
		}
	}

	if portErr != nil {
		return portErr
	}

	d.SetId(portId)
	port.WaitForPortProvisioning(portId)
	return resourceMegaportPortRead(d, m)
}

func resourceMegaportPortRead(d *schema.ResourceData, m interface{}) error {
	port := m.(*terraform_utility.MegaportClient).Port
	portDetails, retrievalErr := port.GetPortDetails(d.Id())

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("uid", portDetails.UID)
	d.Set("port_name", portDetails.Name)
	d.Set("type", portDetails.Type)
	d.Set("provisioning_status", portDetails.ProvisioningStatus)
	d.Set("create_date", portDetails.CreateDate)
	d.Set("created_by", portDetails.CreatedBy)
	d.Set("port_speed", portDetails.PortSpeed)
	d.Set("live_date", portDetails.LiveDate)
	d.Set("market_code", portDetails.Market)
	d.Set("location_id", portDetails.LocationID)
	d.Set("marketplace_visibility", portDetails.MarketplaceVisibility)
	d.Set("company_name", portDetails.CompanyName)
	d.Set("term", portDetails.ContractTermMonths)
	d.Set("lag_primary", portDetails.LAGPrimary)
	d.Set("lag_id", portDetails.LAGID)
	d.Set("locked", portDetails.Locked)
	d.Set("admin_locked", portDetails.AdminLocked)
	d.Set("diversity_zone", portDetails.DiversityZone)

	return nil
}

func resourceMegaportPortUpdate(d *schema.ResourceData, m interface{}) error {
	port := m.(*terraform_utility.MegaportClient).Port

	if d.HasChange("port_name") || d.HasChange("marketplace_visibility") {
		_, nameErr := port.ModifyPort(d.Id(),
			d.Get("port_name").(string),
			"",
			d.Get("marketplace_visibility").(bool))

		if nameErr != nil {
			return nameErr
		}
	}

	if d.HasChange("locked") {
		if d.Get("locked").(bool) {
			lockStatus, lockErr := port.LockPort(d.Id())

			if lockErr != nil {
				return lockErr
			} else {
				if !lockStatus {
					return errors.New(PortNotLockedError)
				}
			}
		} else {
			unlockStatus, unlockErr := port.UnlockPort(d.Id())

			if unlockErr != nil {
				return unlockErr
			} else {
				if !unlockStatus {
					return errors.New(PortNotUnlockedError)
				}
			}
		}
	}

	return resourceMegaportPortRead(d, m)
}

func resourceMegaportPortDelete(d *schema.ResourceData, m interface{}) error {
	port := m.(*terraform_utility.MegaportClient).Port

	// we don't want to automatically delete resources as this has physical ramifications and can't be undone.
	if m.(*terraform_utility.MegaportClient).DeletePorts {

		deleteSuccess, deleteError := port.DeletePort(d.Id(), true)

		if !deleteSuccess {
			return errors.New(fmt.Sprintf("Error deleting resource %s: %s", d.Id(), deleteError))
		}

	} else {

		cancelSuccess, cancelError := port.DeletePort(d.Id(), false)

		if !cancelSuccess {
			return errors.New(fmt.Sprintf("Error cancelling resource %s: %s", d.Id(), cancelError))
		}
	}

	return nil
}
