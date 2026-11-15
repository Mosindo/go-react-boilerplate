import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { LinearGradient } from "expo-linear-gradient";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/profile";
import { queryKeys } from "../../api/queryKeys";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { errorMessage, isApiError } from "../../lib/errors";
import { genderLabels } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { celebratedMatches } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { ActionSheet, ErrorState, IconButton, Loading, PhotoImage, Text, toast } from "../../ui";
import { useMyProfile } from "../profile/hooks";
import { BlockConfirm, ReportSheet } from "../safety/SafetyActions";
import { useSwipe } from "./hooks";
import { Fact, facts, factStyles, InterestChips } from "./ProfileFacts";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ route, navigation }: Props) {
  const { userId, source } = route.params;
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const { width } = useWindowDimensions();
  const queryClient = useQueryClient();
  const me = useMyProfile();
  const swipe = useSwipe();
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const [blocking, setBlocking] = useState(false);

  const cached = queryClient.getQueryData<{ profiles: ProfileCard[] }>(queryKeys.discovery)?.profiles.find((p) => p.userId === userId);
  const profile = useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => profileApi.view(userId), initialData: cached });

  const canSwipe = source === "discovery" || source === "likes";
  const mine = me.data?.profile.interests.map((i) => i.id) ?? [];

  const act = (action: SwipeAction) => {
    swipe.mutate(
      { userId, action },
      {
        onSuccess: (result) => {
          void queryClient.invalidateQueries({ queryKey: queryKeys.discovery });
          if (result.matched && result.match) {
            celebratedMatches.add(result.match.id);
            toast(`C'est un match avec ${result.match.user.firstName} !`, "success");
            navigation.replace("Chat", { conversationId: result.match.conversationId });
          } else {
            navigation.goBack();
          }
        },
        onError: (e) => {
          if (isApiError(e, "already_swiped")) navigation.goBack();
          else toast(errorMessage(e), "error");
        }
      }
    );
  };

  const close = () => navigation.goBack();
  const removed = () => navigation.popToTop();

  if (profile.isPending) return <Loading />;
  if (profile.isError || !profile.data) {
    return (
      <View style={[styles.flex, { backgroundColor: colors.background, paddingTop: insets.top }]}>
        <IconButton icon="close" label="Fermer" onPress={close} style={styles.floatingClose} />
        <ErrorState message={errorMessage(profile.error)} />
      </View>
    );
  }

  const card = profile.data;
  const photoHeight = Math.min(width * 1.25, 620);
  return (
    <View style={[styles.flex, { backgroundColor: colors.background }]} testID="profile-detail">
      <ScrollView contentContainerStyle={{ paddingBottom: canSwipe ? 120 : spacing.xxl }} showsVerticalScrollIndicator={false}>
        <PhotoImage
          photo={card.photos[0]}
          style={{ width: "100%", height: photoHeight }}
          priority="high"
          accessibilityLabel={`Photo principale de ${card.firstName}`}
        />
        <View style={styles.body}>
          <View style={styles.nameRow}>
            <Text variant="display">{card.firstName}</Text>
            <Text variant="display" style={styles.age}>
              {card.age}
            </Text>
          </View>
          {card.isDemo ? (
            <Text variant="micro" color="accent">
              PROFIL DÉMO — DONNÉES FICTIVES
            </Text>
          ) : null}
          <View style={factStyles.row}>
            {card.gender ? <Fact icon="person-outline" label={genderLabels[card.gender]} /> : null}
            {facts(card).map((f) => (
              <Fact key={f.label} icon={f.icon} label={f.label} />
            ))}
          </View>
          {card.bio ? (
            <View style={[styles.block, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <Text variant="micro" color="textSubtle">
                À PROPOS
              </Text>
              <Text>{card.bio}</Text>
            </View>
          ) : null}
          {card.interests.length ? (
            <View style={[styles.block, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <Text variant="micro" color="textSubtle">
                CENTRES D’INTÉRÊT
              </Text>
              <InterestChips card={card} mine={mine} />
            </View>
          ) : null}
          {card.photos.slice(1).map((p, i) => (
            <PhotoImage
              key={p.id}
              photo={p}
              style={[styles.photo, { height: photoHeight * 0.85 }]}
              accessibilityLabel={`Photo ${i + 2} de ${card.firstName}`}
            />
          ))}
          <View style={styles.safety}>
            <Text variant="label" color="textMuted" onPress={() => setReporting(true)} accessibilityRole="button" testID="profile-report">
              Signaler {card.firstName}
            </Text>
            <Text variant="label" color="textMuted" onPress={() => setBlocking(true)} accessibilityRole="button" testID="profile-block">
              Bloquer
            </Text>
          </View>
        </View>
      </ScrollView>
      <View style={[styles.topBar, { top: insets.top + spacing.xs }]}>
        <IconButton icon="chevron-down" label="Fermer le profil" onPress={close} background="surface" />
        <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} background="surface" />
      </View>
      {canSwipe ? (
        <View style={[styles.bottomBar, { paddingBottom: insets.bottom + spacing.md }]}>
          <LinearGradient
            colors={["transparent", colors.background]}
            locations={[0, 0.45]}
            style={StyleSheet.absoluteFill}
            pointerEvents="none"
          />
          <IconButton
            icon="close"
            label={`Passer ${card.firstName}`}
            size={60}
            tone="danger"
            background="surface"
            onPress={() => act("pass")}
            disabled={swipe.isPending}
            testID="detail-pass"
          />
          <IconButton
            icon="heart"
            label={`Aimer ${card.firstName}`}
            size={60}
            tone="onPrimary"
            background="primary"
            onPress={() => act("like")}
            disabled={swipe.isPending}
            testID="detail-like"
          />
        </View>
      ) : null}
      <ActionSheet
        visible={menu}
        onClose={() => setMenu(false)}
        actions={[
          { label: `Signaler ${card.firstName}`, onPress: () => setReporting(true), destructive: true },
          { label: "Bloquer", onPress: () => setBlocking(true), destructive: true }
        ]}
      />
      <ReportSheet
        visible={reporting}
        userId={userId}
        name={card.firstName}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          removed();
        }}
      />
      <BlockConfirm
        visible={blocking}
        userId={userId}
        name={card.firstName}
        onClose={() => setBlocking(false)}
        onDone={() => {
          setBlocking(false);
          removed();
        }}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  body: { padding: spacing.lg, gap: spacing.md },
  nameRow: { flexDirection: "row", alignItems: "baseline", gap: spacing.sm },
  age: { fontWeight: "400" },
  block: { borderWidth: 1, borderRadius: radii.lg, padding: spacing.md, gap: spacing.sm },
  photo: { width: "100%", borderRadius: radii.lg },
  safety: { flexDirection: "row", justifyContent: "center", gap: spacing.xl, paddingVertical: spacing.lg },
  topBar: { position: "absolute", left: spacing.md, right: spacing.md, flexDirection: "row", justifyContent: "space-between" },
  floatingClose: { margin: spacing.md },
  bottomBar: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl,
    paddingTop: spacing.md
  }
});
