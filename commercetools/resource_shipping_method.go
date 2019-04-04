package commercetools

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/labd/commercetools-go-sdk/commercetools"
)

func resourceShippingMethod() *schema.Resource {
	return &schema.Resource{
		Create: resourceShippingMethodCreate,
		Read:   resourceShippingMethodRead,
		Update: resourceShippingMethodUpdate,
		Delete: resourceShippingMethodDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceShippingMethodCreate(d *schema.ResourceData, m interface{}) error {
	client := getClient(m)
	var shippingMethod *commercetools.ShippingMethod
	emptyTaxCategory := commercetools.TaxCategoryReference{}

	draft := &commercetools.ShippingMethodDraft{
		Key:         d.Get("key").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		TaxCategory: &emptyTaxCategory,
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error

		shippingMethod, err = client.ShippingMethodCreate(draft)
		if err != nil {
			if ctErr, ok := err.(commercetools.ErrorResponse); ok {
				if _, ok := ctErr.Errors[0].(commercetools.InvalidJSONInputError); ok {
					return resource.NonRetryableError(ctErr)
				}
			} else {
				log.Printf("[DEBUG] Received error: %s", err)
			}
			return resource.RetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if shippingMethod == nil {
		log.Fatal("No shipping method created?")
	}

	d.SetId(shippingMethod.ID)
	d.Set("version", shippingMethod.Version)

	return resourceShippingMethodRead(d, m)
}

func resourceShippingMethodRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] Reading shipping method from commercetools, with shippingMethod id: %s", d.Id())

	client := getClient(m)

	shippingMethod, err := client.ShippingMethodGetByID(d.Id())

	if err != nil {
		if ctErr, ok := err.(commercetools.ErrorResponse); ok {
			if ctErr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if shippingMethod == nil {
		log.Print("[DEBUG] No shipping method found")
		d.SetId("")
	} else {
		log.Print("[DEBUG] Found following shipping method:")
		log.Print(stringFormatObject(shippingMethod))

		d.Set("version", shippingMethod.Version)
		d.Set("key", shippingMethod.Key)
		d.Set("name", shippingMethod.Name)
		d.Set("description", shippingMethod.Description)
	}

	return nil
}

func resourceShippingMethodUpdate(d *schema.ResourceData, m interface{}) error {
	ctMutexKV.Lock(d.Id())
	defer ctMutexKV.Unlock(d.Id())

	client := getClient(m)
	shippingMethod, err := client.ShippingMethodGetByID(d.Id())
	if err != nil {
		return err
	}

	input := &commercetools.ShippingMethodUpdateInput{
		ID:      d.Id(),
		Version: shippingMethod.Version,
		Actions: []commercetools.ShippingMethodUpdateAction{},
	}

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		input.Actions = append(
			input.Actions,
			&commercetools.ShippingMethodChangeNameAction{Name: newName})
	}

	if d.HasChange("key") {
		newKey := d.Get("key").(string)
		input.Actions = append(
			input.Actions,
			&commercetools.ShippingMethodSetKeyAction{Key: newKey})
	}

	if d.HasChange("description") {
		newDescription := d.Get("description").(string)
		input.Actions = append(
			input.Actions,
			&commercetools.ShippingMethodSetDescriptionAction{Description: newDescription})
	}

	log.Printf(
		"[DEBUG] Will perform update operation with the following actions:\n%s",
		stringFormatActions(input.Actions))

	_, err = client.ShippingMethodUpdate(input)
	if err != nil {
		if ctErr, ok := err.(commercetools.ErrorResponse); ok {
			log.Printf("[DEBUG] %v: %v", ctErr, stringFormatErrorExtras(ctErr))
		}
		return err
	}

	return resourceShippingMethodRead(d, m)
}

func resourceShippingMethodDelete(d *schema.ResourceData, m interface{}) error {
	client := getClient(m)

	ctMutexKV.Lock(d.Id())
	defer ctMutexKV.Unlock(d.Id())

	shippingMethod, err := client.ShippingMethodGetByID(d.Id())
	if err != nil {
		return err
	}

	_, err = client.ShippingMethodDeleteByID(d.Id(), shippingMethod.Version)
	if err != nil {
		return err
	}

	return nil
}