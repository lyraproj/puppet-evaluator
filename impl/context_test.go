package impl_test

import "github.com/puppetlabs/go-evaluator/eval"
import (
	_ "github.com/puppetlabs/go-evaluator/pcore"
	"fmt"
)

func ExampleContext_parseType2() {
	c := eval.Puppet.RootContext()
	ts := c.ParseType2(`TypeSet[{pcore_uri => 'http://puppet.com/2016.1/pcore', pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'Genesis::Digitalocean', version => '1.0.0', types => {Droplet => Resource{attributes => {'ensure' => {'type' => Optional[Enum['present', 'absent']], 'value' => undef}, 'region' => String, 'image' => String, 'size' => String, 'name_prefix' => String, 'ssh_key_names' => Array[String], 'tags' => {'type' => Array[String], 'value' => []}, 'ipv6' => {'type' => Boolean, 'value' => false}, 'backups' => {'type' => Boolean, 'value' => false}, 'private_networking' => {'type' => Boolean, 'value' => false}, 'monitoring' => {'type' => Boolean, 'value' => false}, 'user_data' => Optional[String], 'droplet_id' => Optional[Integer], 'droplet_name' => Optional[String], 'memory' => Optional[Integer], 'vcpus' => Optional[Integer], 'disk' => Optional[Integer], 'image_id' => Optional[Integer], 'image_name' => Optional[String], 'image_distribution' => Optional[String], 'ip_address' => Optional[String], 'netmask' => Optional[String], 'gateway' => Optional[String], 'type' => Optional[String]}}}}]`)
	c.AddTypes(ts)
	if tp, ok := eval.Load(c, eval.NewTypedName(eval.TYPE, `Genesis::Digitalocean::Droplet`)); ok {
		fmt.Println(tp)
	}
	// Output: Object[{name => 'Genesis::Digitalocean::Droplet', parent => Resource, attributes => {'ensure' => {'type' => Optional[Enum['present', 'absent']], 'value' => undef}, 'region' => String, 'image' => String, 'size' => String, 'name_prefix' => String, 'ssh_key_names' => Array[String], 'tags' => {'type' => Array[String], 'value' => []}, 'ipv6' => {'type' => Boolean, 'value' => false}, 'backups' => {'type' => Boolean, 'value' => false}, 'private_networking' => {'type' => Boolean, 'value' => false}, 'monitoring' => {'type' => Boolean, 'value' => false}, 'user_data' => {'type' => Optional[String], 'value' => undef}, 'droplet_id' => {'type' => Optional[Integer], 'value' => undef}, 'droplet_name' => {'type' => Optional[String], 'value' => undef}, 'memory' => {'type' => Optional[Integer], 'value' => undef}, 'vcpus' => {'type' => Optional[Integer], 'value' => undef}, 'disk' => {'type' => Optional[Integer], 'value' => undef}, 'image_id' => {'type' => Optional[Integer], 'value' => undef}, 'image_name' => {'type' => Optional[String], 'value' => undef}, 'image_distribution' => {'type' => Optional[String], 'value' => undef}, 'ip_address' => {'type' => Optional[String], 'value' => undef}, 'netmask' => {'type' => Optional[String], 'value' => undef}, 'gateway' => {'type' => Optional[String], 'value' => undef}, 'type' => {'type' => Optional[String], 'value' => undef}}}]
}
