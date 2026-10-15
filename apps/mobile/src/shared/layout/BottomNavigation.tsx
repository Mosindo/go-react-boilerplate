import React from "react";
import { Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { BottomTabBarProps } from "@react-navigation/bottom-tabs";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../../theme";
import { CountBadge, Text, spacing } from "../ui";

type IconName = React.ComponentProps<typeof Ionicons>["name"];

export type TabMeta = {
  label: string;
  icon: IconName;
  iconActive: IconName;
};

export type TabBadges = Partial<Record<string, number>>;

/**
 * Custom tab bar: large touch targets, labels always visible (accessibility),
 * and unread counters on the tabs that need them.
 */
export function BottomNavigation({
  badges,
  meta,
  state,
  navigation
}: BottomTabBarProps & { meta: Record<string, TabMeta>; badges: TabBadges }) {
  const palette = useTheme();
  const insets = useSafeAreaInsets();

  return (
    <View
      accessibilityRole="tablist"
      style={{
        flexDirection: "row",
        backgroundColor: palette.surface,
        borderTopWidth: 1,
        borderTopColor: palette.border,
        paddingBottom: Math.max(insets.bottom, spacing.sm),
        paddingTop: spacing.xs
      }}
    >
      {state.routes.map((route, index) => {
        const focused = state.index === index;
        const tab = meta[route.name];
        if (!tab) {
          return null;
        }
        const color = focused ? palette.primary : palette.textSubtle;
        const count = badges[route.name] ?? 0;
        return (
          <Pressable
            accessibilityLabel={count > 0 ? `${tab.label}, ${count} non lu${count > 1 ? "s" : ""}` : tab.label}
            accessibilityRole="tab"
            accessibilityState={{ selected: focused }}
            key={route.key}
            onPress={() => {
              const event = navigation.emit({ type: "tabPress", target: route.key, canPreventDefault: true });
              if (!focused && !event.defaultPrevented) {
                navigation.navigate(route.name);
              }
            }}
            style={{ flex: 1, minHeight: 52, alignItems: "center", justifyContent: "center", gap: 2 }}
            testID={`tab-${route.name}`}
          >
            <View>
              <Ionicons color={color} name={focused ? tab.iconActive : tab.icon} size={25} />
              {count > 0 ? <CountBadge count={count} style={{ position: "absolute", top: -6, right: -12 }} /> : null}
            </View>
            <Text style={{ color }} variant="caption" weight={focused ? "semibold" : "medium"}>
              {tab.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}
