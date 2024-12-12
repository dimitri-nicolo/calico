import { PluginConfigDescriptor, PluginInitializerContext } from '@kbn/core/server';
import { configSchema, ConfigSchema } from '../common';

//  This exports static code and TypeScript types,
//  as well as, Kibana Platform `plugin()` initializer.

export async function plugin(initializerContext: PluginInitializerContext<ConfigSchema>) {
  const { TigeraPlugin } = await import('./plugin');
  return new TigeraPlugin(initializerContext);
}

export const config: PluginConfigDescriptor<ConfigSchema> = {
  schema: configSchema,
  exposeToBrowser: {
    licenseEdition: true,
  },
};

