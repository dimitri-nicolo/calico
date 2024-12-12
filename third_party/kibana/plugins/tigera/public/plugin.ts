import { PluginInitializerContext, CoreSetup, CoreStart, Plugin } from '@kbn/core/public';
import { ConfigSchema } from '../common';

export class TigeraPlugin implements Plugin {
  constructor(_initializerContext: PluginInitializerContext<ConfigSchema>) {}

  public setup(core: CoreSetup) {}

  public start(core: CoreStart) {}

  public stop() {}
}
