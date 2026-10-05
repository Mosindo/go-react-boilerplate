import { StyleSheet, View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { radii, spacing, useTheme } from "../../theme";

export function Card({ style, ...props }: ViewProps & { style?: StyleProp<ViewStyle> }) {
  const { colors } = useTheme();
  return <View {...props} style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }, style]} />;
}

const styles = StyleSheet.create({ card: { borderRadius: radii.lg, borderWidth: 1, padding: spacing.lg } });
