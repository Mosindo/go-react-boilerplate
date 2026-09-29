import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Card } from "../ui/Card";
import { Loader } from "../ui/Loader";
import { Text } from "../ui/Text";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type LoadingViewProps = {
  fullScreen?: boolean;
  label?: string;
  message?: string;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  tone?: "default" | "muted";
};

const makeStyles = (t: Theme) => ({
  fullScreen: { flex: 1, alignItems: "center" as const, justifyContent: "center" as const },
  inline: { width: "100%" as const },
  card: {
    width: "100%" as const,
    maxWidth: 420,
    alignSelf: "center" as const,
    alignItems: "center" as const,
    gap: t.spacing.sm
  },
  message: { textAlign: "center" as const }
});

export function LoadingView({
  fullScreen = false,
  label = "Loading...",
  message,
  style,
  testID,
  tone = "muted"
}: LoadingViewProps) {
  const styles = useThemedStyles(makeStyles);
  const content = (
    <Card padding="lg" style={styles.card} variant="muted">
      <Loader label={label} />
      {message ? (
        <Text style={styles.message} tone={tone}>
          {message}
        </Text>
      ) : null}
    </Card>
  );

  return (
    <View style={[fullScreen ? styles.fullScreen : styles.inline, style]} testID={testID}>
      {content}
    </View>
  );
}
