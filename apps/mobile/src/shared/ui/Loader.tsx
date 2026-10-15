import React from "react";
import { ActivityIndicator, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { spacing } from "./tokens";

export type LoaderProps = {
  label?: string;
  fullScreen?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Loader({ fullScreen = false, label, style }: LoaderProps) {
  const palette = useTheme();
  return (
    <View
      accessibilityLabel={label ?? "Chargement"}
      accessibilityRole="progressbar"
      style={[
        { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
        fullScreen ? { flex: 1 } : null,
        style
      ]}
    >
      <ActivityIndicator color={palette.primary} size="large" />
      {label ? (
        <Text tone="muted" variant="label">
          {label}
        </Text>
      ) : null}
    </View>
  );
}
