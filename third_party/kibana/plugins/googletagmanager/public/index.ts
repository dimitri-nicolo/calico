import { PluginInitializerContext } from '@kbn/core/public';
import { GoogletagmanagerPlugin } from './plugin';
import { ConfigSchema} from '../common';

// This exports static code and TypeScript types,
// as well as, Kibana Platform `plugin()` initializer.
export function plugin(initializerContext: PluginInitializerContext<ConfigSchema>) {
  return new GoogletagmanagerPlugin(initializerContext);
}

