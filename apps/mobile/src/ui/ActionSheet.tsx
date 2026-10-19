import React, { createContext, useCallback, useContext, useMemo, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { MIN_TOUCH, radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

export type SheetAction = { label: string; destructive?: boolean; onPress: () => void };
type Sheet = { title?: string; actions: SheetAction[] };
type ActionSheetContextValue = { show: (sheet: Sheet) => void };

const ActionSheetContext = createContext<ActionSheetContextValue>({ show: () => undefined });

/**
 * Cross-platform action sheet. `Alert.alert` is limited to three buttons on
 * Android, which is not enough for the photo / chat / profile menus.
 */
export function ActionSheetProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const [sheet, setSheet] = useState<Sheet | null>(null);
  const show = useCallback((next: Sheet) => setSheet(next), []);
  const close = useCallback(() => setSheet(null), []);
  const value = useMemo(() => ({ show }), [show]);

  return (
    <ActionSheetContext.Provider value={value}>
      {children}
      <Modal
        transparent
        animationType="fade"
        visible={sheet !== null}
        onRequestClose={close}
        accessibilityViewIsModal
      >
        <Pressable
          style={[styles.backdrop, { backgroundColor: colors.overlay }]}
          onPress={close}
          accessibilityLabel="Fermer le menu"
          accessibilityRole="button"
        >
          <SafeAreaView edges={["bottom"]} style={styles.sheetWrap}>
            <View style={[styles.sheet, { backgroundColor: colors.surface }]} testID="action-sheet">
              {sheet?.title ? (
                <Text variant="caption" tone="muted" bold style={styles.title}>
                  {sheet.title}
                </Text>
              ) : null}
              {sheet?.actions.map((action) => (
                <Pressable
                  key={action.label}
                  accessibilityRole="button"
                  accessibilityLabel={action.label}
                  style={[styles.action, { borderColor: colors.border }]}
                  onPress={() => {
                    close();
                    action.onPress();
                  }}
                >
                  <Text bold tone={action.destructive ? "danger" : "default"}>
                    {action.label}
                  </Text>
                </Pressable>
              ))}
              <Pressable
                accessibilityRole="button"
                accessibilityLabel="Annuler"
                style={styles.action}
                onPress={close}
              >
                <Text tone="muted" bold>
                  Annuler
                </Text>
              </Pressable>
            </View>
          </SafeAreaView>
        </Pressable>
      </Modal>
    </ActionSheetContext.Provider>
  );
}

export function useActionSheet(): ActionSheetContextValue {
  return useContext(ActionSheetContext);
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheetWrap: { paddingHorizontal: spacing.md },
  sheet: { borderRadius: radius.lg, overflow: "hidden", marginBottom: spacing.sm },
  title: { textAlign: "center", paddingVertical: spacing.md },
  action: {
    minHeight: MIN_TOUCH + 4,
    alignItems: "center",
    justifyContent: "center",
    borderTopWidth: StyleSheet.hairlineWidth
  }
});
