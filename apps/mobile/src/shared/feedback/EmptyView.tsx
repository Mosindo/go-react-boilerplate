import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../../theme";
import { Button } from "../ui/Button";
import { Text } from "../ui/Text";

type Props = {
  icon?: React.ComponentProps<typeof Ionicons>["name"];
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyView({ icon = "heart-outline", title, message, actionLabel, onAction, testID }: Props) {
  const theme = useTheme();
  return (
    <View style={styles.center} testID={testID ?? "empty-view"}>
      <Ionicons name={icon} size={48} color={theme.primary} />
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" center>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }
});
