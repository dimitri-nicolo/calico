
export default function (kibana) {
  return new kibana.Plugin({
   uiExports: {
     app: {
        title: 'tigera_customization',
        order: -100,
        description: 'Tigera Customization',
        main: 'plugins/tigera_customization/index.js',
        hidden: true
     }
    }
  });
};
