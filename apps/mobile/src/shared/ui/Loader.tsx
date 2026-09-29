import React from "react";
import {
  ActivityIndicator,
  View,
  type ActivityIndicatorProps,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { Text } from "./Text";
import { useTheme, type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type LoaderProps = {
  fullScreen?: boolean;
  label?: string;
  size?: ActivityIndicatorProps["size"];
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  inline: { alignItems: "center" as const, justifyContent: "center" as const, gap: t.spacing.sm },
  fullScreen: {
    flex: 1,
    alignItems: "center" as const,
    justifyContent: "center" as const,
    gap: t.spacing.md
  }
});

export function Loader({ fullScreen = false, label, size = "large", style }: LoaderProps) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  return (
    <View
      accessibilityLabel={label ?? "Loading"}
      accessibilityRole="progressbar"
      style={[fullScreen ? styles.fullScreen : styles.inline, style]}
    >
      <ActivityIndicator color={theme.colors.primary} size={size} />
      {label ? (
        <Text tone="muted" variant="label">
          {label}
        </Text>
      ) : null}
    </View>
  );
}
