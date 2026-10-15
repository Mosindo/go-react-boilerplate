import React, { useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { blockUser } from "../api/platform";
import { ActionSheet } from "../components/ActionSheet";
import { ProfileBody } from "../components/ProfileBody";
import { queryKeys, useMyProfile, useProfile } from "../hooks/queries";
import { errorCodeOf, errorMessage } from "../i18n/errors";
import { confirm } from "../lib/confirm";
import { announceProfileRemoved } from "../lib/removedProfiles";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Button, spacing } from "../shared/ui";
import { useTheme } from "../theme";

export default function ProfileDetailScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { params } = useRoute<RouteProp<RootStackParamList, "ProfileDetail">>();
  const queryClient = useQueryClient();
  const query = useProfile(params.userId);
  const mine = useMyProfile();
  const [menuOpen, setMenuOpen] = useState(false);
  const card = query.data;

  const shared = new Set(mine.data?.profile?.interests.map((i) => i.slug));
  const name = card?.firstName ?? "cette personne";

  const block = async () => {
    try {
      await blockUser(params.userId);
      announceProfileRemoved(params.userId);
      queryClient.removeQueries({ queryKey: queryKeys.profile(params.userId) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.chats });
      await queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
      showSuccess(`${name} est bloqué(e).`);
      navigation.goBack();
    } catch (error) {
      showError(errorMessage(error));
    }
  };

  const actions = [
    {
      label: "Signaler",
      destructive: true,
      onPress: () => navigation.navigate("Report", { userId: params.userId, name })
    },
    {
      label: `Bloquer ${name}`,
      destructive: true,
      onPress: () =>
        confirm({
          title: `Bloquer ${name} ?`,
          message: "Vous ne vous verrez plus mutuellement et toute conversation sera supprimée.",
          confirmLabel: "Bloquer",
          destructive: true,
          onConfirm: () => void block()
        })
    }
  ];

  const unavailable = query.isError && errorCodeOf(query.error) === "profile_not_found";

  return (
    <SafeAreaLayout edges={["top", "left", "right", "bottom"]}>
      <View style={{ paddingHorizontal: spacing.lg }}>
        <Header
          onBack={() => navigation.goBack()}
          right={
            card ? (
              <Pressable
                accessibilityLabel="Plus d'options"
                accessibilityRole="button"
                hitSlop={8}
                onPress={() => setMenuOpen(true)}
                style={{ width: 44, height: 44, alignItems: "center", justifyContent: "center" }}
                testID="profile-menu"
              >
                <Ionicons color={palette.text} name="ellipsis-horizontal" size={24} />
              </Pressable>
            ) : undefined
          }
          title={card?.firstName ?? "Profil"}
        />
      </View>

      {query.isLoading ? <LoadingView fullScreen /> : null}
      {unavailable ? (
        <ErrorView message="Ce profil n'est plus disponible." onRetry={() => navigation.goBack()} title="Profil introuvable" />
      ) : query.isError ? (
        <ErrorView error={query.error} onRetry={() => void query.refetch()} />
      ) : null}

      {card ? (
        <ScrollView
          contentContainerStyle={{ padding: spacing.lg, paddingBottom: spacing.xxxl, gap: spacing.lg }}
          showsVerticalScrollIndicator={false}
          testID="profile-detail"
        >
          <ProfileBody
            age={card.age}
            bio={card.bio}
            city={card.city}
            distanceKm={card.distanceKm}
            firstName={card.firstName}
            gender={card.gender}
            highlightSlugs={shared}
            interests={card.interests}
            photos={card.photos}
          />
          {params.conversationId ? (
            <Button
              fullWidth
              label={`Écrire à ${card.firstName}`}
              onPress={() =>
                navigation.navigate("Thread", { conversationId: params.conversationId!, peerId: params.userId, peerName: card.firstName })
              }
              size="lg"
            />
          ) : null}
        </ScrollView>
      ) : null}

      <ActionSheet actions={actions} onClose={() => setMenuOpen(false)} title={card?.firstName} visible={menuOpen} />
    </SafeAreaLayout>
  );
}
