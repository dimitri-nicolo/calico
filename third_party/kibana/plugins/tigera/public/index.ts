import './index.scss';

import { PluginInitializerContext } from '@kbn/core/public';
import { ConfigSchema } from '../common';
import { TigeraPlugin } from './plugin';

// This exports static code and TypeScript types,
// as well as, Kibana Platform `plugin()` initializer.
export function plugin(initializerContext: PluginInitializerContext<ConfigSchema>) {
  return new TigeraPlugin(initializerContext);
}
