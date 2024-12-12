export const PLUGIN_ID = 'tigera';
export const PLUGIN_NAME = 'tigera';


import { schema, TypeOf } from '@kbn/config-schema';

export const configSchema = schema.object({
  enabled: schema.boolean({ defaultValue: true }),
  licenseEdition: schema.oneOf([
    schema.literal('enterpriseEdition'),
    schema.literal('cloudEdition'),
  ]),
});

export type ConfigSchema = TypeOf<typeof configSchema>;

