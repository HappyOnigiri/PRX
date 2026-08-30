import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

interface LicensePackage {
  license: string;
  name: string;
  repository: string | null;
  source: string | null;
  version: string;
}

const licenseReportUrl = `${import.meta.env.BASE_URL}oss-licenses.json`;

function isLicensePackage(value: unknown): value is LicensePackage {
  if (typeof value !== "object" || value === null) return false;
  const packageInfo = value as Partial<LicensePackage>;
  return (
    typeof packageInfo.license === "string" &&
    typeof packageInfo.name === "string" &&
    (typeof packageInfo.repository === "string" ||
      packageInfo.repository === null) &&
    (typeof packageInfo.source === "string" || packageInfo.source === null) &&
    typeof packageInfo.version === "string"
  );
}

function externalUrl(value: string | null) {
  if (!value) return null;
  const normalized = value.replace(/^git\+/, "");
  try {
    const url = new URL(normalized);
    return url.protocol === "http:" || url.protocol === "https:"
      ? url.toString()
      : null;
  } catch {
    return null;
  }
}

export function LicensesSettingsPanel() {
  const { t } = useTranslation();
  const [packages, setPackages] = useState<LicensePackage[]>();
  const [loadError, setLoadError] = useState(false);

  useEffect(() => {
    let active = true;
    void fetch(licenseReportUrl)
      .then((response) => {
        if (!response.ok) throw new Error("License report request failed");
        return response.json() as Promise<unknown>;
      })
      .then((value) => {
        if (!Array.isArray(value)) throw new Error("License report is invalid");
        const report = value.filter(isLicensePackage).sort((left, right) => {
          const nameOrder = left.name.localeCompare(right.name);
          return nameOrder || left.version.localeCompare(right.version);
        });
        if (active) setPackages(report);
      })
      .catch(() => {
        if (active) setLoadError(true);
      });
    return () => {
      active = false;
    };
  }, []);

  if (loadError)
    return <p className="settings-empty">{t("settings.licenses.error")}</p>;
  if (!packages)
    return <p className="settings-empty">{t("settings.licenses.loading")}</p>;
  if (packages.length === 0)
    return <p className="settings-empty">{t("settings.licenses.empty")}</p>;

  return (
    <ul className="settings-license-list">
      {packages.map((packageInfo) => {
        const href =
          externalUrl(packageInfo.repository) ??
          externalUrl(packageInfo.source);
        const label = `${packageInfo.name}@${packageInfo.version}`;
        return (
          <li className="settings-license-row" key={label}>
            {href ? (
              <a
                className="settings-license-name"
                href={href}
                target="_blank"
                rel="noreferrer"
              >
                {label}
              </a>
            ) : (
              <span className="settings-license-name">{label}</span>
            )}
            <span className="settings-license-id">{packageInfo.license}</span>
          </li>
        );
      })}
    </ul>
  );
}
