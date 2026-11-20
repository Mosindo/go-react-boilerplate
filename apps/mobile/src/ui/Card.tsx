import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme";
import { Text } from "./Text";

export function Card({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  const { colors, radii, spacing } = useTheme();
  return (
    <View
      style={[
        { backgroundColor: colors.surface, borderRadius: radii.lg, borderWidth: 1, borderColor: colors.border, padding: spacing.lg, gap: spacing.sm },
        style
      ]}
    >
      {children}
    </View>
  );
}

export function SectionTitle({ children }: { children: string }) {
  return (
    <Text tone="subtle" variant="eyebrow">
      {children}
    </Text>
  );
}

export function Badge({ label, tone = "primary" }: { label: string; tone?: "primary" | "accent" }) {
  const { colors, radii } = useTheme();
  return (
    <View
      style={{
        alignSelf: "flex-start",
        paddingHorizontal: 8,
        paddingVertical: 3,
        borderRadius: radii.pill,
        backgroundColor: tone === "primary" ? colors.primarySoft : colors.accentSoft
      }}
    >
      <Text style={{ color: tone === "primary" ? colors.primary : colors.accent, fontSize: 11, fontWeight: "700" }} variant="caption">
        {label}
      </Text>
    </View>
  );
}

export function CountBadge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View
      style={{
        minWidth: 20,
        height: 20,
        paddingHorizontal: 6,
        borderRadius: 10,
        backgroundColor: colors.primary,
        alignItems: "center",
        justifyContent: "center"
      }}
    >
      <Text style={{ color: colors.onPrimary, fontSize: 11, fontWeight: "800" }} variant="caption">
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}
