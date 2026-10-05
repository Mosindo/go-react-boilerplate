import { Ionicons } from "@expo/vector-icons";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Badge } from "./Badge";

export function IconButton({
  icon,
  label,
  onPress,
  size = 24,
  color,
  badge,
  style,
  testID
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  badge?: number;
  style?: StyleProp<ViewStyle>;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      testID={testID}
      onPress={onPress}
      hitSlop={8}
      style={({ pressed }) => [styles.button, { opacity: pressed ? 0.6 : 1 }, style]}
    >
      <Ionicons name={icon} size={size} color={color ?? colors.text} />
      {badge ? (
        <View style={styles.badge}>
          <Badge count={badge} />
        </View>
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  badge: { position: "absolute", top: 0, right: 0 }
});
