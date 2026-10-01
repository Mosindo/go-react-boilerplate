import React, { useEffect } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Text, radii, shadows, spacing, useTheme } from "../ui";
import { clearGlobalError, useGlobalFeedback } from "./store";

/** Non-blocking banner for API/network problems; auto-dismisses after 5 s. */
export function Toast() {
  const { error } = useGlobalFeedback();
  const { colors } = useTheme();

  useEffect(() => {
    if (!error) {
      return undefined;
    }
    const id = setTimeout(clearGlobalError, 5000);
    return () => clearTimeout(id);
  }, [error]);

  if (!error) {
    return null;
  }
  return (
    <SafeAreaView edges={["top"]} pointerEvents="box-none" style={styles.wrap}>
      <Pressable
        accessibilityLabel={`${error}. Toucher pour fermer.`}
        accessibilityRole="alert"
        onPress={clearGlobalError}
        style={[styles.toast, shadows.floating, { backgroundColor: colors.text }]}
      >
        <View>
          <Text style={{ color: colors.background }} variant="label" weight="semibold">
            {error}
          </Text>
        </View>
      </Pressable>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", top: 0, left: 0, right: 0, paddingHorizontal: spacing.lg, zIndex: 50 },
  toast: { borderRadius: radii.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, marginTop: spacing.sm }
});
