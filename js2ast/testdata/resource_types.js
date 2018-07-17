// Nested types used by the resources
create_types({
  'Genesis::Aws::IpRange': {
    cidr_ip: 'String',
    description: 'String'
  },

  'Genesis::Aws::IpPermission': {
    from_port: 'Integer',
    to_port: 'Integer',
    ip_protocol: 'String',
    ip_ranges: 'Array[Genesis::Aws::IpRange]'
  },

  'Genesis::Aws::Route': {
    destination_cidr_block: 'String',
    gateway_id: 'String'
  },

  'Genesis::Aws::TagSpecification': {
    resource_type: 'String',
    tags: 'Hash[String,String]'
  },

  'Genesis::Aws::GroupIdentifier': {
    group_id: 'String'
  },
});

// Create the resource types. This is for test only. In normal cases, those
// types are provided by registerd plugins
create_resource_types({
  'Genesis::Aws::ImportKeypair': {
    ensure: 'Enum[absent,present]',
    region: 'String',
    public_key_material: 'String',
    fingerprint: {
      type: 'String',
      value: 'FAKE_FINGERPRINT'
    },
    key_name: {
      type: 'String',
      value: 'FAKE_KEY_NAME'
    }
  },

  'Genesis::Aws::Vpc': {
    ensure: 'Enum[absent,present]',
    region: 'String',
    cidr_block: 'String',
    tags: 'Hash[String,String]',
    enable_dns_hostnames: 'Boolean',
    enable_dns_support: 'Boolean',
    vpc_id: {
      type: 'String',
      value: 'FAKE_VPC_ID'
    }
  },

  'Genesis::Aws::SecurityGroup': {
    ensure: 'Enum[absent,present]',
    region: 'String',
    tags: 'Hash[String,String]',
    description: 'String',
    vpc_id: 'String',
    ip_permissions: 'Array[Genesis::Aws::IpPermission]',
    group_id: {
      type: 'String',
      value: 'FAKE_GROUP_ID'
    }
  },

  'Genesis::Aws::Subnet': {
    ensure: 'Enum[absent,present]',
    region: 'String',
    tags: 'Hash[String,String]',
    cidr_block: 'String',
    vpc_id: 'String',
    map_public_ip_on_launch: 'Boolean',
    subnet_id: {
      type: 'String',
      value: 'FAKE_SUBNET_ID'
    }
  },

  'Genesis::Aws::InternetGateway': {
    ensure: 'Enum[absent,present]',
    region: 'String',
    tags: 'Hash[String,String]',
    internet_gateway_id: {
      type: 'String',
      value: 'FAKE_GATEWAY_ID'
    }
  },

  'Genesis::Aws::RouteTable': {
    ensure: 'Enum[absent,present]',
    region: 'String',
    vpc_id: 'String',
    subnet_id: 'String',
    tags: 'Hash[String,String]',
    routes: 'Array[Genesis::Aws::Route]',
    route_table_id: {
      type: 'String',
      value: 'FAKE_ROUTE_TABLE_ID'
    }
  },

  'Genesis::Aws::Instance': {
    region: 'String',
    image_id: 'String',
    instance_type: 'String',
    key_name: 'String',
    subnet_id: 'String',
    tag_specifications: 'Array[Genesis::Aws::TagSpecification]',
    security_groups: 'Array[Genesis::Aws::GroupIdentifier]',
    instance_id: {
      type: 'String',
      value: 'FAKE_INSTANCE_ID'
    },
    private_ip_address: {
      type: 'String',
      value: 'FAKE_PRIVATE_IP'
    },
    public_ip_address: {
      type: 'String',
      value: 'FAKE_PUBLIC_IP'
    },
  },

  'Genesis::Ssh::Exec': {
    ensure: 'Enum[absent,present]',
    user: 'String',
    host: 'String',
    port: 'Integer',
    command: 'String'
  },
});
