import React, { useLayoutEffect, useState } from "react";
import { Pressable, ScrollView, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { safetyApi } from "../api/endpoints";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { ReportSheet } from "../components/ReportSheet";
import { errorMessage, keys, usePublicProfile } from "../hooks/queries";
import { distanceLabel, genderLabel } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { discoverStore } from "../state/discoverStore";
import {
  Chip,
  ErrorView,
  LoadingView,
  Screen,
  Sheet,
  SheetAction,
  Text,
  radii,
  spacing,
  useTheme,
  useToast
} from "../ui";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, showActions = false } = route.params;
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = usePublicProfile(userId);
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const profile = query.data;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile?.displayName ?? "",
      headerRight: () => (
        <Pressable
          accessibilityLabel="Plus d'options"
          accessibilityRole="button"
          onPress={() => setMenu(true)}
          style={styles.menuButton}
          testID="profile-menu"
        >
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      )
    });
  }, [navigation, profile?.displayName, colors.text]);

  const block = async () => {
    setMenu(false);
    try {
      await safetyApi.block(userId);
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.blocked });
      discoverStore.refresh();
      toast.show("Utilisateur bloqué.");
      navigation.popToTop();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const decide = (action: "like" | "pass") => {
    if (!profile) return;
    void discoverStore.swipe({ ...profile }, action);
    navigation.goBack();
  };

  if (query.isLoading) return <LoadingView />;
  if (query.isError || !profile)
    return <ErrorView message="Ce profil n'est plus disponible." onRetry={() => navigation.goBack()} />;

  const distance = distanceLabel(profile.distanceKm);

  return (
    <Screen padded={false} testID="profile-detail">
      <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
        <View style={styles.photo}>
          <PhotoCarousel photos={profile.photos} rounded={false} />
        </View>
        <View style={styles.content}>
          <Text variant="title">
            {profile.displayName}, {profile.age}
          </Text>
          <Text tone="muted">{[genderLabel(profile.gender), profile.city, distance].filter(Boolean).join(" · ")}</Text>
          {profile.bio ? <Text>{profile.bio}</Text> : null}
          {profile.interests.length > 0 ? (
            <View style={styles.chips}>
              {profile.interests.map((interest) => (
                <Chip key={interest.slug} label={interest.label} />
              ))}
            </View>
          ) : null}
        </View>
      </ScrollView>

      {showActions ? (
        <View style={[styles.bar, { backgroundColor: colors.background, borderColor: colors.border }]}>
          <Pressable
            accessibilityLabel="Passer"
            accessibilityRole="button"
            onPress={() => decide("pass")}
            style={[styles.round, { borderColor: colors.border, backgroundColor: colors.surface }]}
            testID="detail-pass"
          >
            <Ionicons color={colors.danger} name="close" size={30} />
          </Pressable>
          <Pressable
            accessibilityLabel="J'aime"
            accessibilityRole="button"
            onPress={() => decide("like")}
            style={[styles.round, { borderColor: colors.border, backgroundColor: colors.surface }]}
            testID="detail-like"
          >
            <Ionicons color={colors.primary} name="heart" size={30} />
          </Pressable>
        </View>
      ) : null}

      <Sheet onClose={() => setMenu(false)} title={profile.displayName} visible={menu}>
        <SheetAction
          label="Signaler"
          onPress={() => {
            setMenu(false);
            setReporting(true);
          }}
          testID="menu-report"
        />
        <SheetAction danger label="Bloquer" onPress={block} testID="menu-block" />
      </Sheet>
      <ReportSheet
        name={profile.displayName}
        onClose={() => setReporting(false)}
        onDone={() => navigation.popToTop()}
        userId={userId}
        visible={reporting}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  scroll: { paddingBottom: 110 },
  photo: { width: "100%", aspectRatio: 4 / 5 },
  content: { padding: spacing.lg, gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  menuButton: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  bar: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl,
    paddingVertical: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  round: {
    width: 64,
    height: 64,
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  }
});
