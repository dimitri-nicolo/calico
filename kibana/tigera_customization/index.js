// https://discuss.elastic.co/t/make-variable-configurable-for-plugin-and-reference-in-code/139414/4
// https://discuss.elastic.co/t/reading-config-env-variable-inside-kibana-plugin/167643

export default function (kibana) {
  return new kibana.Plugin({
    id: 'tigera_customization',
    configPrefix: 'tigera_customization',
    uiExports: {
      app: {
        title: 'tigera_customization',
        order: -100,
        description: 'Tigera Customization',
        main: 'plugins/tigera_customization/index.js',
        hidden: true,
      },
      hacks: [
        'plugins/tigera_customization/index_hack',
      ],
      injectDefaultVars(server) {
        const config = server.config();
        return {
          gtm: config.get('tigera_customization.gtm'),
        };
      }
    },
    config(Joi) {
      return Joi.object({
        enabled: Joi.boolean().default(true),
        gtm: Joi.string().default('disable'),
      }).default();
    },
  });
};
