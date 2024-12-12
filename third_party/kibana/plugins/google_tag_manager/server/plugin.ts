import {
  PluginInitializerContext,
  CoreSetup,
  CoreStart,
  Plugin,
  Logger
} from '../../../src/core/server';
import { schema } from '@kbn/config-schema';
import { ConfigSchema } from '../common';

export class GoogleTagManagerPlugin implements Plugin {
  private readonly logger: Logger;

  constructor(initializerContext: PluginInitializerContext) {
    this.logger = initializerContext.logger.get();
  }

  public setup(core: CoreSetup) {
    this.logger.info('googleTagManager: Setup');
    return {};
  }

  public start(core: CoreStart) {
    this.logger.debug('googleTagManager: Started');
    return {};
  }

  public stop() {}
}
