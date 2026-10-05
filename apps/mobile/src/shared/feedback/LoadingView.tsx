import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Loader } from "../ui/Loader";
import { Text } from "../ui/Text";

export function LoadingView({ label }: { label?: string }) {
  return (
    <View style={styles.box} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <Loader size="large" />
      {label ? (
        <Text tone="muted" center>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({ box: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl } });
