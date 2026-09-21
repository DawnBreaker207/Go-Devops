import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
// Never import 'antd/dist/reset.css' here: it is @imported with a layer in
// index.css. A JS import becomes unlayered CSS beating layered utilities on
// form elements (`input,button,...{color:inherit}`).
import '@/locales/i18n';
import '@/index.css';
import App from '@/App';

const container = document.getElementById('root');
if (!container) {
  throw new Error('Root element #root not found');
}

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>
);
