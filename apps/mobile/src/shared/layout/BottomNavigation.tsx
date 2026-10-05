import { Ionicons } from "@expo/vector-icons";
import type { BottomTabBarProps } from "@react-navigation/bottom-tabs";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useUnreadMessages } from "../../hooks/queries";
import { spacing, useTheme } from "../../theme";
import { Badge } from "../ui/Badge";
import { Text } from "../ui/Text";

const TABS: Record<string, { label: string; icon: keyof typeof Ionicons.glyphMap; active: keyof typeof Ionicons.glyphMap }> = {
  Discover: { label: "Découvrir", icon: "flame-outline", active: "flame" },
  Messages: { label: "Messages", icon: "chatbubbles-outline", active: "chatbubbles" },
  Me: { label: "Profil", icon: "person-outline", active: "person" }
};

/** Custom tab bar: themed, accessible (selected state, badge announced) and constrained on wide screens. */
export function BottomNavigation({ state, navigation }: BottomTabBarProps) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const unread = useUnreadMessages();

  return (
    <View
      style={[
        styles.bar,
        { backgroundColor: colors.surface, borderTopColor: colors.border, paddingBottom: Math.max(insets.bottom, spacing.sm) }
      ]}
      accessibilityRole="tablist"
    >
      {state.routes.map((route, index) => {
        const tab = TABS[route.name];
        const focused = state.index === index;
        const color = focused ? colors.primary : colors.textMuted;
        return (
          <Pressable
            key={route.key}
            testID={`tab-${route.name}`}
            accessibilityRole="tab"
            accessibilityState={{ selected: focused }}
            accessibilityLabel={route.name === "Messages" && unread > 0 ? `${tab.label}, ${unread} non lus` : tab.label}
            onPress={() => {
              const event = navigation.emit({ type: "tabPress", target: route.key, canPreventDefault: true });
              if (!focused && !event.defaultPrevented) navigation.navigate(route.name);
            }}
            style={styles.item}
          >
            <View>
              <Ionicons name={focused ? tab.active : tab.icon} size={26} color={color} />
              {route.name === "Messages" && unread > 0 ? (
                <View style={styles.badge}>
                  <Badge count={unread} />
                </View>
              ) : null}
            </View>
            <Text variant="caption" style={{ color, fontWeight: focused ? "700" : "500" }}>
              {tab.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  bar: { flexDirection: "row", borderTopWidth: 1, paddingTop: spacing.sm },
  item: { flex: 1, alignItems: "center", gap: 2, minHeight: 52 },
  badge: { position: "absolute", top: -6, right: -14 }
});
