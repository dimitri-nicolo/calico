export const PLUGIN_ID = 'googleTagManager';
export const PLUGIN_NAME = 'googleTagManager';

import { schema, TypeOf } from '@kbn/config-schema';

export const configSchema = schema.object({
  container: schema.string(),
  enabled: schema.boolean({ defaultValue: true }),
});

export type ConfigSchema = TypeOf<typeof configSchema>;
