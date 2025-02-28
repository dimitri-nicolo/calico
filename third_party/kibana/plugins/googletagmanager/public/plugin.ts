import { PluginInitializerContext, CoreSetup, CoreStart, Plugin } from '@kbn/core/public';
import { ConfigSchema } from '../common';

export class GoogletagmanagerPlugin implements Plugin {
  private config: ConfigSchema;
  private logger: any;

  constructor(private readonly initializerContext: PluginInitializerContext<ConfigSchema>) {
    this.config = initializerContext.config.get();
    this.logger = initializerContext.logger.get();
  }

  public setup(core: CoreSetup) {
    const googleTagId = this.config.container;

    if (!googleTagId) {
      this.logger.info('Google Tag Manager ID is not defined');
      return;
    }

    this.initGoogletagmanager(googleTagId);

    return {};
  }

  public start(core: CoreStart) {
    // Implement any start logic if needed
  }

  public stop() {
    // Cleanup if necessary
  }

  private initGoogletagmanager(containerId: string) {
    (function (w: any, d: any, s, l, i) {
      w[l] = w[l] || [];
      w[l].push({ 'gtm.start': new Date().getTime(), event: 'gtm.js' });
      const f = d.getElementsByTagName(s)[0];
      const j = d.createElement(s);
      const dl = l !== 'dataLayer' ? '&l=' + l : '';
      j.async = true;
      j.src = 'https://www.googletagmanager.com/gtm.js?id=' + i + dl;
      f.parentNode.insertBefore(j, f);
    })(window, document, 'script', 'dataLayer', containerId);
  }
}
