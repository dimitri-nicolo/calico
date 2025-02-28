export const PLUGIN_ID = 'googletagmanager';
export const PLUGIN_NAME = 'googletagmanager';

import { schema, TypeOf } from '@kbn/config-schema';

export const configSchema = schema.object({
  container: schema.string(),
  enabled: schema.boolean({ defaultValue: true }),
});

export type ConfigSchema = TypeOf<typeof configSchema>;
