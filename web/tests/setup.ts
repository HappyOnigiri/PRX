import "@testing-library/jest-dom/vitest";
import "../src/i18n";

window.scrollTo = () => undefined;

// jsdom は Pointer Events のキャプチャを実装していない。rail のリサイズは
// setPointerCapture を呼ぶので、何もしない実装を置く。
Element.prototype.setPointerCapture = () => undefined;
Element.prototype.releasePointerCapture = () => undefined;
