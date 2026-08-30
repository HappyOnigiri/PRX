import type { LucideIcon, LucideProps } from "lucide-react";
import type { ButtonHTMLAttributes, Ref } from "react";

type IconButtonVariant = "primary" | "secondary" | "quiet" | "danger";
type IconButtonSize = "standard" | "compact";

type IconProps = Omit<
  LucideProps,
  "aria-hidden" | "focusable" | "size" | "strokeWidth"
> &
  Record<`data-${string}`, string | number | undefined>;

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: LucideIcon;
  label: string;
  iconOnly?: boolean;
  ref?: Ref<HTMLButtonElement>;
  size?: IconButtonSize;
  variant?: IconButtonVariant;
  iconProps?: IconProps;
}

export function IconButton({
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
    className,
  ]
    .filter(Boolean)
    .join(" ");
  const ariaLabel = buttonProps["aria-label"] ?? label;
  const iconSize = size === "compact" ? 14 : 16;

  return (
    <button
      {...buttonProps}
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
