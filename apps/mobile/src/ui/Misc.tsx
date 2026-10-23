import React from "react";
import { Switch, View } from "react-native";
import { t } from "../i18n/fr";
import { createStyles, useTheme } from "../theme/ThemeProvider";
import { ScalePressable } from "./Pressable";
import { Text } from "./Text";

const useStyles = createStyles(({ colors, spacing, radius, minTouch }) => ({
  chip: {
    minHeight: minTouch,
    paddingHorizontal: spacing.lg,
    borderRadius: radius.pill,
    borderWidth: 1.5,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    alignItems: "center",
    justifyContent: "center"
  },
  chipOn: { backgroundColor: colors.primary, borderColor: colors.primary },
  chipSmall: { minHeight: 32, paddingHorizontal: spacing.md },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    backgroundColor: colors.accent,
    alignItems: "center",
    justifyContent: "center"
  },
  row: {
    minHeight: minTouch + 12,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm
  },
  rowSeparated: { borderBottomWidth: 1, borderBottomColor: colors.border },
  stepper: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepButton: {
    width: minTouch,
    height: minTouch,
    borderRadius: minTouch / 2,
    backgroundColor: colors.surfaceAlt,
    alignItems: "center",
    justifyContent: "center"
  },
  tag: {
    borderRadius: radius.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: 6,
    backgroundColor: colors.surfaceAlt
  },
  tagShared: { backgroundColor: colors.highlight },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
}));

export function Chip({
  label,
  selected,
  onPress,
  testID
}: {
  label: string;
  selected: boolean;
  onPress: () => void;
  testID?: string;
}) {
  const styles = useStyles();
  return (
    <ScalePressable
      accessibilityLabel={label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected }}
      onPress={onPress}
      testID={testID}
    >
      <View style={[styles.chip, selected ? styles.chipOn : null]}>
        <Text variant="label" tone={selected ? "onPrimary" : "default"}>
          {label}
        </Text>
      </View>
    </ScalePressable>
  );
}

/** Read-only tag; `shared` highlights common interests. */
export function Tag({ label, shared }: { label: string; shared?: boolean }) {
  const styles = useStyles();
  return (
    <View style={[styles.tag, shared ? styles.tagShared : null]}>
      <Text variant="caption" style={shared ? { color: "#2B1228" } : undefined}>
        {label}
      </Text>
    </View>
  );
}

export function TagWrap({ children }: { children: React.ReactNode }) {
  return <View style={useStyles().wrap}>{children}</View>;
}

export function CountBadge({ count }: { count: number }) {
  const styles = useStyles();
  if (count <= 0) return null;
  return (
    <View accessibilityLabel={t.a11y.tabBadge(count)} style={styles.badge}>
      <Text variant="caption" tone="onAccent" style={{ fontWeight: "700" }}>
        {count > 99 ? "99+" : String(count)}
      </Text>
    </View>
  );
}

export function SwitchRow({
  label,
  hint,
  value,
  onChange,
  testID
}: {
  label: string;
  hint?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  testID?: string;
}) {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <View style={[styles.row, styles.rowSeparated]}>
      <View style={{ flex: 1 }}>
        <Text variant="label">{label}</Text>
        {hint ? (
          <Text variant="caption" tone="muted">
            {hint}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        onValueChange={onChange}
        thumbColor={colors.surface}
        trackColor={{ false: colors.border, true: colors.primary }}
        testID={testID}
        value={value}
      />
    </View>
  );
}

export function Stepper({
  label,
  value,
  unit,
  onChange,
  min,
  max,
  step = 1,
  testID
}: {
  label: string;
  value: number;
  unit: string;
  onChange: (value: number) => void;
  min: number;
  max: number;
  step?: number;
  testID?: string;
}) {
  const styles = useStyles();
  const change = (delta: number) => onChange(Math.min(max, Math.max(min, value + delta)));
  return (
    <View style={[styles.row, styles.rowSeparated]}>
      <Text variant="label" style={{ flex: 1 }}>
        {label}
      </Text>
      <View style={styles.stepper}>
        <ScalePressable
          accessibilityLabel={`${t.preferences.decrease} : ${label}`}
          accessibilityRole="button"
          disabled={value <= min}
          onPress={() => change(-step)}
          testID={testID ? `${testID}-dec` : undefined}
        >
          <View style={[styles.stepButton, value <= min ? { opacity: 0.4 } : null]}>
            <Text variant="heading">−</Text>
          </View>
        </ScalePressable>
        <Text
          accessibilityLiveRegion="polite"
          variant="label"
          style={{ minWidth: 64, textAlign: "center" }}
        >
          {value} {unit}
        </Text>
        <ScalePressable
          accessibilityLabel={`${t.preferences.increase} : ${label}`}
          accessibilityRole="button"
          disabled={value >= max}
          onPress={() => change(step)}
          testID={testID ? `${testID}-inc` : undefined}
        >
          <View style={[styles.stepButton, value >= max ? { opacity: 0.4 } : null]}>
            <Text variant="heading">+</Text>
          </View>
        </ScalePressable>
      </View>
    </View>
  );
}

/** Tappable settings row with chevron. */
export function LinkRow({
  label,
  onPress,
  testID,
  tone = "default",
  hint
}: {
  label: string;
  onPress: () => void;
  testID?: string;
  tone?: "default" | "danger";
  hint?: string;
}) {
  const styles = useStyles();
  return (
    <ScalePressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
    >
      <View style={[styles.row, styles.rowSeparated]}>
        <View style={{ flex: 1 }}>
          <Text variant="label" tone={tone === "danger" ? "danger" : "default"}>
            {label}
          </Text>
          {hint ? (
            <Text variant="caption" tone="muted">
              {hint}
            </Text>
          ) : null}
        </View>
        <Text tone="muted">›</Text>
      </View>
    </ScalePressable>
  );
}
