import type { LucideIcon, LucideProps } from "lucide-react";
import type { ButtonHTMLAttributes, Ref } from "react";

type IconButtonVariant =
  "primary" | "secondary" | "quiet" | "danger" | "affirm";
type IconButtonSize = "standard" | "compact";

type IconProps = Omit<
  LucideProps,
  "aria-hidden" | "focusable" | "size" | "strokeWidth"
> &
  Record<`data-${string}`, string | number | undefined>;

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: LucideIcon;
  label: string;
  busy?: boolean;
  iconOnly?: boolean;
  ref?: Ref<HTMLButtonElement>;
  size?: IconButtonSize;
  variant?: IconButtonVariant;
  iconProps?: IconProps;
}

export function IconButton({
  busy = false,
  className,
  icon: Icon,
  iconOnly = false,
  iconProps,
  label,
  ref,
  size = "standard",
  title,
  type = "button",
  variant = "secondary",
  ...buttonProps
}: IconButtonProps) {
  const classes = [
    "icon-button",
    `icon-button-${size}`,
    `icon-button-${variant}`,
    iconOnly ? "icon-button-only" : "icon-button-with-label",
    busy ? "icon-button-busy" : undefined,
    className,
  ]
    .filter(Boolean)
    .join(" ");
  const ariaLabel = buttonProps["aria-label"] ?? label;
  const iconSize = size === "compact" ? 14 : 16;

  return (
    <button
      {...buttonProps}
      aria-busy={buttonProps["aria-busy"] ?? (busy || undefined)}
      aria-label={ariaLabel}
      className={classes}
      ref={ref}
      title={title ?? label}
      type={type}
    >
      <Icon
        {...iconProps}
        aria-hidden="true"
        focusable="false"
        size={iconSize}
        strokeWidth={1.75}
      />
      {!iconOnly && <span className="icon-button-label">{label}</span>}
    </button>
  );
}
