import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";

export function Avatar({ name, path, size = 52 }: { name: string; path?: string; size?: number }) {
  const { colors } = useTheme();
  if (path) {
    return <RemoteImage path={path} accessibilityLabel={`Photo de ${name}`} style={{ width: size, height: size, borderRadius: size / 2 }} />;
  }
  return (
    <View style={[styles.fallback, { width: size, height: size, borderRadius: size / 2, backgroundColor: colors.primarySoft }]}>
      <Text variant="bodyStrong" tone="primary">
        {name.trim().charAt(0).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({ fallback: { alignItems: "center", justifyContent: "center" } });
