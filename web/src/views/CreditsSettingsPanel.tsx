import { useTranslation } from "react-i18next";

const reactFlowLicense = `MIT License

Copyright (c) 2019-2025 webkid GmbH

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`;

export function CreditsSettingsPanel() {
  const { t } = useTranslation();
  return (
    <>
      <p className="dialog-lead">{t("settings.credits.description")}</p>
      <section
        className="settings-section"
        aria-labelledby="settings-react-flow"
      >
        <header>
          <p className="section-label">
            {t("settings.credits.reactFlowLabel")}
          </p>
          <h3 id="settings-react-flow">
            {t("settings.credits.reactFlowName")}
          </h3>
        </header>
        <p className="settings-credit-description">
          {t("settings.credits.reactFlowDescription")}
        </p>
        <p className="settings-credit-links">
          <a href="https://reactflow.dev" target="_blank" rel="noreferrer">
            {t("settings.credits.website")}
          </a>
          <a
            href="https://github.com/xyflow/xyflow/blob/main/packages/react/LICENSE"
            target="_blank"
            rel="noreferrer"
          >
            {t("settings.credits.licenseSource")}
          </a>
        </p>
        <pre className="settings-license">{reactFlowLicense}</pre>
      </section>
    </>
  );
}
