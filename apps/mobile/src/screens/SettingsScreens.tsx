import React, { useState } from "react";
import { FlatList, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { absoluteUrl } from "../api/client";
import { friendlyError } from "../api/errors";
import { savePreferences, savePrivacy, unblockUser } from "../api/platform";
import type { BlockedUser, Preferences } from "../api/types";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesFields, SectionCard } from "../components/profileFields";
import { ToggleRow } from "../components/Switch";
import { queryKeys, useBlockedUsers, useMyProfile } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Avatar, Button, Card, Notice, Text, spacing } from "../shared/ui";
import { showAlert } from "../lib/alert";

type StackProps<K extends keyof RootStackParamList> = NativeStackScreenProps<RootStackParamList, K>;

function Loading() {
  return (
    <SafeAreaLayout>
      <LoadingView fullScreen label="Chargement…" />
    </SafeAreaLayout>
  );
}

// --------------------------------------------------------------------- photos

export function PhotosScreen() {
  const me = useMyProfile();
  if (me.isLoading) {
    return <Loading />;
  }
  if (!me.data) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger vos photos." onAction={() => void me.refetch()} />
      </SafeAreaLayout>
    );
  }
  return (
    <SafeAreaLayout>
      <ScrollView contentContainerStyle={styles.scroll} testID="screen-photos">
        <Text variant="title" weight="bold">
          Mes photos
        </Text>
        <PhotoManager photos={me.data.photos} />
        {me.data.photos.length === 0 ? (
          <Notice description="Sans photo, votre profil n'est pas proposé aux autres." title="Ajoutez au moins une photo" tone="warning" />
        ) : null}
      </ScrollView>
    </SafeAreaLayout>
  );
}

// ---------------------------------------------------------------- preferences

export function PreferencesScreen({ navigation }: StackProps<"Preferences">) {
  const queryClient = useQueryClient();
  const me = useMyProfile();
  const [draft, setDraft] = useState<Preferences | null>(null);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: savePreferences,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.me });
      navigation.goBack();
    },
    onError: (e) => setError(friendlyError(e))
  });

  if (me.isLoading) {
    return <Loading />;
  }
  const current = draft ?? me.data?.preferences;
  if (!current) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger vos préférences." onAction={() => void me.refetch()} />
      </SafeAreaLayout>
    );
  }
  return (
    <SafeAreaLayout>
      <ScrollView contentContainerStyle={styles.scroll} testID="screen-preferences">
        <Text variant="title" weight="bold">
          Préférences de rencontre
        </Text>
        <SectionCard>
          <PreferencesFields onChange={setDraft} value={current} />
        </SectionCard>
        {error ? <Notice title={error} tone="danger" /> : null}
        <Button
          disabled={current.interestedIn.length === 0}
          label="Enregistrer"
          loading={save.isPending}
          onPress={() => {
            setError(null);
            save.mutate(current);
          }}
          testID="save-preferences"
        />
      </ScrollView>
    </SafeAreaLayout>
  );
}

// -------------------------------------------------------------------- privacy

export function PrivacyScreen() {
  const queryClient = useQueryClient();
  const me = useMyProfile();
  const [error, setError] = useState<string | null>(null);

  const update = useMutation({
    mutationFn: savePrivacy,
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.me, data);
    },
    onError: (e) => setError(friendlyError(e))
  });

  if (me.isLoading) {
    return <Loading />;
  }
  const profile = me.data?.profile;
  if (!profile) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger vos réglages." onAction={() => void me.refetch()} />
      </SafeAreaLayout>
    );
  }
  return (
    <SafeAreaLayout>
      <ScrollView contentContainerStyle={styles.scroll} testID="screen-privacy">
        <Text variant="title" weight="bold">
          Confidentialité
        </Text>
        <SectionCard>
          <ToggleRow
            description="Désactivé, votre profil n'est plus proposé à de nouvelles personnes. Vos matchs et conversations restent actifs."
            disabled={update.isPending}
            label="Apparaître dans la découverte"
            onChange={(discoverable) => {
              setError(null);
              update.mutate({ discoverable });
            }}
            testID="toggle-discoverable"
            value={profile.discoverable}
          />
          <ToggleRow
            description="Les autres voient une distance approximative (jamais votre position exacte)."
            disabled={update.isPending}
            label="Afficher ma distance"
            onChange={(showDistance) => {
              setError(null);
              update.mutate({ showDistance });
            }}
            testID="toggle-distance"
            value={profile.showDistance}
          />
        </SectionCard>
        {error ? <Notice title={error} tone="danger" /> : null}
        <Text tone="muted" variant="caption">
          Votre adresse e-mail et votre date de naissance ne sont jamais visibles par les autres membres.
        </Text>
      </ScrollView>
    </SafeAreaLayout>
  );
}

// -------------------------------------------------------------------- blocked

export function BlockedScreen() {
  const queryClient = useQueryClient();
  const blocked = useBlockedUsers();
  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
    },
    onError: (e) => showAlert("Action impossible", friendlyError(e))
  });

  return (
    <SafeAreaLayout>
      <View style={styles.scroll} testID="screen-blocked">
        <Text variant="title" weight="bold">
          Personnes bloquées
        </Text>
        {blocked.isLoading ? <LoadingView label="Chargement…" /> : null}
        {blocked.isError ? <ErrorView message="Impossible de charger la liste." onAction={() => void blocked.refetch()} /> : null}
        {blocked.data ? (
          <FlatList<BlockedUser>
            contentContainerStyle={styles.list}
            data={blocked.data}
            keyExtractor={(item) => item.profile.id}
            ListEmptyComponent={<EmptyView message="Vous n'avez bloqué personne." title="Liste vide" />}
            renderItem={({ item }) => (
              <Card padding="md" style={styles.blockedRow}>
                <Avatar name={item.profile.firstName} uri={item.profile.photos[0] ? absoluteUrl(item.profile.photos[0].url) : null} />
                <Text style={styles.grow} weight="bold">
                  {item.profile.firstName}
                </Text>
                <Button
                  label="Débloquer"
                  loading={unblock.isPending && unblock.variables === item.profile.id}
                  onPress={() => unblock.mutate(item.profile.id)}
                  size="sm"
                  variant="outline"
                />
              </Card>
            )}
          />
        ) : null}
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  scroll: { padding: spacing.xl, gap: spacing.lg, paddingBottom: spacing.xxxl },
  list: { gap: spacing.sm, paddingTop: spacing.md },
  blockedRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  grow: { flex: 1 }
});
