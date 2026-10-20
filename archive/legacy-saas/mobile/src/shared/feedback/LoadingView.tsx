import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Card, Loader, Text, colors, spacing } from "../ui";

type LoadingViewProps = {
  fullScreen?: boolean;
  label?: string;
  message?: string;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  tone?: "default" | "muted";
};

export function LoadingView({
  fullScreen = false,
  label = "Loading...",
  message,
  style,
  testID,
  tone = "muted"
}: LoadingViewProps) {
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

  if (fullScreen) {
    return (
      <View style={[styles.fullScreen, style]} testID={testID}>
        {content}
      </View>
    );
  }

  return (
    <View style={[styles.inline, style]} testID={testID}>
      {content}
    </View>
  );
}

const styles = StyleSheet.create({
  fullScreen: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  inline: {
    width: "100%"
  },
  card: {
    width: "100%",
    maxWidth: 420,
    alignSelf: "center",
    alignItems: "center",
    gap: spacing.sm,
    borderColor: colors.border
  },
  message: {
    textAlign: "center"
  }
});
