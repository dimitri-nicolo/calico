import chrome from 'ui/chrome';
import { initGoogleTagManager } from 'plugins/tigera_customization/googleTagManager';

const gtm = chrome.getInjected('gtm');

if (gtm === 'enable') {
    initGoogleTagManager('GTM-TCNXTCJ');
}
