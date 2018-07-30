var region = 'eu-west-1';
var tags = {
    'created_by':'admin@example.com',
    'department':'engineering',
    'project':'incubator',
    'lifetime':'1h'
};

var install_nginx = 'sudo apt -y update\n' +
  'sudo apt -y install nginx\n' +
  'sudo rm -rf /var/www/html/index.html\n' +
  'echo "<html><head><title>Installed by Genesis</title></head><body><h1>INSTALLED BY GENESIS</h1></body></html>" | sudo tee /var/www/html/index.html';

var public_key = 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQCX363gh/q6DGSL963/LlYcILkYKtEjrq5Ze4gr1BJdY0pqLMIKFt/VMJ5UTyx85N4Chjb/jEQhZzlWGC1SMsXOQ+EnY72fYrpOV0wZ4VraxZAz3WASikEglHJYALTQtsL8RGPxlBhIv0HpgevBkDlHvR+QGFaEQCaUhXCWDtLWYw== nyx-test-keypair-nopassword';

/*
This example uses named functions tier1, tier2, tierN which are then passed to sequential at the bottom
but it could just as well use anonymous functions, i.e.

sequential(
  function() {
    // this is tier1
  },
  function() {
    // this is tier2
  },
  ...
)
*/

function tier1() {
  resources({
    'genesis::aws::importkeypair': {
      'myapp-keypair': {
        ensure: 'present',
        region: region,
        public_key_material: public_key
      }
    },

    'genesis::aws::vpc': {
      'myapp-vpc': {
        ensure: 'present',
        region: region,
        cidr_block: "192.168.0.0/16",
        tags: tags,
        enable_dns_hostnames: true,
        enable_dns_support: true
      }
    }
  });
}

function tier2() {
  var keypair = resource('genesis::aws::importkeypair[myapp-keypair]');
  var vpc = resource('genesis::aws::vpc[myapp-vpc]');
  console.log("✔ Imported KeyPair: '" + keypair.title + "' with fingerprint: " + keypair.fingerprint);
  console.log("✔ Created VPC: '" + vpc.vpc_id + "' in region '" + region + "'");

  resources({
    'genesis::aws::securitygroup': {
      'myapp-secgroup': {
        ensure: 'present',
        region: vpc.region,
        tags: tags,
        description: 'myapp-secgroup',
        vpc_id: vpc.vpc_id,
        ip_permissions: [
          new Genesis.Aws.Ippermission({
            from_port: 0,
            to_port: 0,
            ip_protocol: '-1',
            ip_ranges: [
              new Genesis.Aws.Iprange({
                cidr_ip: '0.0.0.0/0',
                description: 'any source address'
              })
            ]
          })
        ]
      }
    },
    'genesis::aws::subnet': {
      'myapp-subnet': {
        ensure: 'present',
        region: vpc.region,
        vpc_id: vpc.vpc_id,
        cidr_block: '192.168.1.0/24',
        tags: tags,
        map_public_ip_on_launch: true
      }
    },
    'genesis::aws::internetgateway': {
      'myapp-gateway': {
        ensure: 'present',
        region: vpc.region,
        tags: tags
      }
    }
  });
}

function tier3() {
  var vpc = resource('genesis::aws::vpc[myapp-vpc]');
  var secgroup = resource('genesis::aws::securitygroup[myapp-secgroup]');
  var subnet = resource('genesis::aws::subnet[myapp-subnet]');
  var gateway = resource('genesis::aws::internetgateway[myapp-gateway]');
  console.log("✔ Created SecurityGroup: '" + secgroup.group_id + "'");
  console.log("✔ Created Subnet: '" + subnet.subnet_id + "'");
  console.log("✔ Created InternetGateway: '" + gateway.internet_gateway_id + "'");

  resources({
    'genesis::aws::routetable': {
      'myapp-routes': {
        ensure: 'present',
        region: region,
        vpc_id: vpc.vpc_id,
        subnet_id: subnet.subnet_id,
        routes: [
          new Genesis.Aws.Route({
            destination_cidr_block: '0.0.0.0/0',
            gateway_id: gateway.internet_gateway_id
          })
        ],
        tags: tags
      }
    }
  })
}

function tier4() {
  var keypair = resource('genesis::aws::importkeypair[myapp-keypair]');
  var routes = resource('genesis::aws::routetable[myapp-routes]');
  var subnet = resource('genesis::aws::subnet[myapp-subnet]');
  var secgroup = resource('genesis::aws::securitygroup[myapp-secgroup]');
  console.log("✔ Created RouteTable '" + routes.route_table_id + "' for Subnet '" + subnet.subnet_id + "'");

  [1,2,3].parallel(function(x) {
    resources({'genesis::aws::instance': new Hash(['myapp-instance-' + x, {
        region: region,
        image_id: 'ami-f90a4880',
        instance_type: 't2.nano',
        key_name: keypair.key_name,
        tag_specifications: [new Genesis.Aws.TagSpecification({resource_type: 'instance', tags: tags})],
        subnet_id: subnet.subnet_id,
        security_groups: [new Genesis.Aws.GroupIdentifier({group_id: secgroup.group_id})]
      }])})
  })
}

function tier5() {
  var instance = resource('genesis::aws::instance[myapp-instance-1]');
  console.log("✔ Created EC2 Instance, ID: '" + instance.instance_id + "', PrivateIP: '" + instance.private_ip_address + "', PublicIP: '" + instance.public_ip_address + "'");

  resources({
    'genesis::ssh::exec': {
      'myapp-sshexec': {
        ensure: 'present',
        host: instance.public_ip_address,
        user: 'ubuntu',
        port: 22,
        command: install_nginx
      }
    }
  })
}

sequential(tier1, tier2, tier3, tier4, tier5);

