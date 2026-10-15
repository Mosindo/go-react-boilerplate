import React from "react";
import { Modal, Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme";
import { Text, radii, spacing } from "../shared/ui";

export type SheetAction = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  disabled?: boolean;
  testID?: string;
};

type ActionSheetProps = {
  visible: boolean;
  title?: string;
  message?: string;
  actions: SheetAction[];
  onClose: () => void;
};

/** Bottom sheet menu. Used instead of Alert because Android alerts cap at three buttons. */
export function ActionSheet({ actions, message, onClose, title, visible }: ActionSheetProps) {
  const palette = useTheme();
  const insets = useSafeAreaInsets();

  return (
    <Modal animationType="fade" onRequestClose={onClose} statusBarTranslucent transparent visible={visible}>
      <Pressable
        accessibilityLabel="Fermer"
        onPress={onClose}
        style={{ flex: 1, backgroundColor: palette.overlay, justifyContent: "flex-end" }}
      >
        <Pressable
          accessibilityViewIsModal
          style={{
            backgroundColor: palette.surface,
            borderTopLeftRadius: radii.xl,
            borderTopRightRadius: radii.xl,
            paddingTop: spacing.lg,
            paddingHorizontal: spacing.lg,
            paddingBottom: Math.max(insets.bottom, spacing.lg),
            gap: spacing.xs
          }}
        >
          <View
            style={{
              alignSelf: "center",
              width: 40,
              height: 4,
              borderRadius: 2,
              backgroundColor: palette.borderStrong,
              marginBottom: spacing.sm
            }}
          />
          {title ? (
            <Text accessibilityRole="header" style={{ textAlign: "center" }} variant="subheading" weight="bold">
              {title}
            </Text>
          ) : null}
          {message ? (
            <Text style={{ textAlign: "center", marginBottom: spacing.sm }} tone="muted" variant="label">
              {message}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              accessibilityRole="button"
              accessibilityState={{ disabled: action.disabled }}
              disabled={action.disabled}
              key={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => ({
                minHeight: 54,
                alignItems: "center",
                justifyContent: "center",
                borderRadius: radii.md,
                backgroundColor: pressed ? palette.surfaceMuted : "transparent",
                opacity: action.disabled ? 0.4 : 1
              })}
              testID={action.testID}
            >
              <Text tone={action.destructive ? "danger" : "default"} weight="semibold">
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable
            accessibilityRole="button"
            onPress={onClose}
            style={({ pressed }) => ({
              minHeight: 54,
              alignItems: "center",
              justifyContent: "center",
              borderRadius: radii.md,
              backgroundColor: pressed ? palette.surfaceMuted : palette.background,
              marginTop: spacing.xs
            })}
          >
            <Text tone="muted" weight="semibold">
              Annuler
            </Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}
