import React, { useLayoutEffect, useState } from "react";
import { Pressable, ScrollView } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { keys } from "../api/keys";
import { fetchPublicProfile } from "../api/profile";
import { ActionSheet, ErrorState, LoadingState, ProfileView, Screen, Text } from "../components";
import { useSafety } from "../hooks/useSafety";
import { errorMessage } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";

export function ProfileDetailScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "ProfileDetail">) {
  const { userId } = route.params;
  const { confirmBlock } = useSafety();
  const [menu, setMenu] = useState(false);
  const query = useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => fetchPublicProfile(userId) });
  const profile = query.data;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile?.firstName ?? "",
      headerRight: () =>
        profile ? (
          <Pressable accessibilityRole="button" accessibilityLabel="More options" onPress={() => setMenu(true)} testID="profile-menu" hitSlop={12} style={{ minWidth: 44, minHeight: 44, alignItems: "center", justifyContent: "center" }}>
            <Text style={{ fontSize: 24 }}>⋯</Text>
          </Pressable>
        ) : null
    });
  }, [navigation, profile]);

  if (query.isPending) return <Screen><LoadingState /></Screen>;
  if (query.isError || !profile) {
    return <Screen><ErrorState message={errorMessage(query.error, "This profile is no longer available.")} onRetry={() => void query.refetch()} /></Screen>;
  }
  return (
    <Screen padded={false} edges={["bottom"]} testID="profile-detail">
      <ScrollView>
        <ProfileView profile={profile} />
      </ScrollView>
      <ActionSheet
        visible={menu}
        title={profile.firstName}
        onClose={() => setMenu(false)}
        actions={[
          { label: "Report", onPress: () => navigation.navigate("Report", { userId, name: profile.firstName }) },
          { label: "Block", destructive: true, onPress: () => confirmBlock(userId, profile.firstName, () => navigation.popToTop()) }
        ]}
      />
    </Screen>
  );
}
