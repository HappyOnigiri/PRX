import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { resources } from "./resources";
import {
  detectDisplayLanguage,
  supportedLanguages,
  writeDisplayLanguage,
  type SupportedLanguage,
} from "./settings";

void i18n.use(initReactI18next).init({
  resources,
  lng: detectDisplayLanguage(),
  fallbackLng: "en",
  supportedLngs: [...supportedLanguages],
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
});

function updateDocument(language: string) {
  document.documentElement.lang = language;
  document.title = i18n.t("app.title");
}

updateDocument(i18n.resolvedLanguage ?? "en");
i18n.on("languageChanged", updateDocument);

export async function setDisplayLanguage(language: SupportedLanguage) {
  writeDisplayLanguage(language);
  await i18n.changeLanguage(language);
}

export default i18n;
