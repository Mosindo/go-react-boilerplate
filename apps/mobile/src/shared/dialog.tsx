import React, { useSyncExternalStore } from "react";
import { Alert, Modal, Platform, Pressable, StyleSheet, View } from "react-native";
import { Text, colors, radii, spacing } from "./ui";

export type DialogButton = { text: string; style?: "default" | "cancel" | "destructive"; onPress?: () => void };

type Dialog = { title: string; message?: string; buttons: DialogButton[] };

let current: Dialog | null = null;
const listeners = new Set<() => void>();

function set(next: Dialog | null) {
  current = next;
  listeners.forEach((l) => l());
}

/**
 * Cross-platform replacement for Alert.alert. Native platforms keep the OS dialog;
 * react-native-web has no Alert implementation, so the web build renders DialogHost instead.
 */
export function showDialog(title: string, message?: string, buttons: DialogButton[] = [{ text: "OK" }]): void {
  if (Platform.OS === "web") {
    set({ title, message, buttons });
    return;
  }
  Alert.alert(title, message, buttons);
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function DialogHost() {
  const dialog = useSyncExternalStore(
    subscribe,
    () => current,
    () => current
  );
  if (!dialog) {
    return null;
  }
  const close = (button?: DialogButton) => {
    set(null);
    button?.onPress?.();
  };
  const cancel = dialog.buttons.find((b) => b.style === "cancel");
  return (
    <Modal animationType="fade" onRequestClose={() => close(cancel)} transparent visible>
      <View style={styles.scrim}>
        <View accessibilityViewIsModal accessibilityRole="alert" style={styles.card} testID="dialog">
          <Text variant="heading" weight="bold">
            {dialog.title}
          </Text>
          {dialog.message ? <Text tone="muted">{dialog.message}</Text> : null}
          <View style={styles.buttons}>
            {dialog.buttons.map((b) => (
              <Pressable accessibilityRole="button" key={b.text} onPress={() => close(b)} style={styles.button} testID={`dialog-${b.text}`}>
                <Text tone={b.style === "destructive" ? "danger" : b.style === "cancel" ? "muted" : "primary"} weight="bold">
                  {b.text}
                </Text>
              </Pressable>
            ))}
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  scrim: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl, backgroundColor: colors.scrim },
  card: { width: "100%", maxWidth: 380, gap: spacing.md, padding: spacing.xl, borderRadius: radii.xl, backgroundColor: colors.background },
  buttons: { gap: spacing.xs, marginTop: spacing.sm },
  button: { paddingVertical: spacing.md, alignItems: "center", borderRadius: radii.md, backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.border }
});
