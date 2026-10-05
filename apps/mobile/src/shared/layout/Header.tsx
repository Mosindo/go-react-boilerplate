import type { ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { IconButton } from "../ui/IconButton";
import { Text } from "../ui/Text";

/** In-screen header: optional back button, title and right-side actions. */
export function Header({ title, subtitle, onBack, right }: { title: string; subtitle?: string; onBack?: () => void; right?: ReactNode }) {
  return (
    <View style={styles.row}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} testID="header-back" /> : null}
      <View style={styles.titles}>
        <Text variant="title" numberOfLines={1}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone="muted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  titles: { flex: 1 },
  right: { flexDirection: "row", alignItems: "center" }
});
