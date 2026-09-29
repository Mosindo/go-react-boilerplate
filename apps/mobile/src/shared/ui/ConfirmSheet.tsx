import React from "react";
import { View } from "react-native";
import { Button } from "./Button";
import { Sheet } from "./Sheet";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type ConfirmSheetProps = {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  actions: { gap: t.spacing.sm }
});

/** Cross-platform replacement for Alert.alert confirmations (works on web too). */
export function ConfirmSheet({
  cancelLabel = "Cancel",
  confirmLabel,
  destructive = false,
  loading = false,
  message,
  onCancel,
  onConfirm,
  testID,
  title,
  visible
}: ConfirmSheetProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <Sheet onClose={onCancel} testID={testID} title={title} visible={visible}>
      <Text tone="muted">{message}</Text>
      <View style={styles.actions}>
        <Button
          label={confirmLabel}
          loading={loading}
          onPress={onConfirm}
          testID={testID ? `${testID}-confirm` : undefined}
          variant={destructive ? "destructive" : "primary"}
        />
        <Button disabled={loading} label={cancelLabel} onPress={onCancel} variant="ghost" />
      </View>
    </Sheet>
  );
}
