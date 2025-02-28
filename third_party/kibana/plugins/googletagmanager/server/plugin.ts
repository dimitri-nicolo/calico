import {
  PluginInitializerContext,
  CoreSetup,
  CoreStart,
  Plugin,
  Logger
} from '../../../src/core/server';
import { schema } from '@kbn/config-schema';
import { ConfigSchema } from '../common';

export class GoogletagmanagerPlugin implements Plugin {
  private readonly logger: Logger;

  constructor(initializerContext: PluginInitializerContext) {
    this.logger = initializerContext.logger.get();
  }

  public setup(core: CoreSetup) {
    this.logger.info('googletagmanager: Setup');
    return {};
  }

  public start(core: CoreStart) {
    this.logger.debug('googletagmanager: Started');
    return {};
  }

  public stop() {}
}
