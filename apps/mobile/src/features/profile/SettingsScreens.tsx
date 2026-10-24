import React, { useEffect, useState } from "react";
import { FlatList, StyleSheet, Switch, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi, safetyApi } from "../../api/endpoints";
import type { Gender, OwnProfile, Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { splitIsoDate } from "../../lib/date";
import { shareApproximateLocation } from "../../lib/location";
import { errorMessage } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import {
  Button,
  EmptyState,
  ErrorState,
  FormError,
  LoadingState,
  Screen,
  Text,
  TextField,
  useToast,
} from "../../ui";
import { DEFAULT_PREFERENCES, GenderPicker, InterestPicker, PreferencesForm } from "./forms";
import { PhotoManager } from "./PhotoManager";
import { useOwnProfile } from "./useOwnProfile";

type Nav = NativeStackNavigationProp<MainStackParamList>;

/** Re-sends the stored profile with some fields changed (birth date can never change). */
function profileInput(
  own: OwnProfile,
  patch: Partial<{
    firstName: string;
    gender: Gender;
    bio: string;
    city: string;
    interests: string[];
    discoverable: boolean;
    showDistance: boolean;
  }>,
) {
  return {
    firstName: own.firstName,
    birthDate: own.birthDate,
    gender: own.gender,
    bio: own.bio,
    city: own.city,
    interests: own.interests.map((i) => i.slug),
    discoverable: own.discoverable,
    showDistance: own.showDistance,
    ...patch,
  };
}

export function EditProfileScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const toast = useToast();
  const { data: own, error, isLoading, refetch } = useOwnProfile();
  const [firstName, setFirstName] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [bio, setBio] = useState("");
  const [city, setCity] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (!own) return;
    setFirstName(own.firstName);
    setGender(own.gender);
    setBio(own.bio);
    setCity(own.city);
    setInterests(own.interests.map((i) => i.slug));
  }, [own]);

  if (isLoading) return <LoadingState />;
  if (error || !own) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const save = async () => {
    if (!firstName.trim() || !gender) {
      setFormError("Prénom et genre sont obligatoires.");
      return;
    }
    setBusy(true);
    setFormError(null);
    try {
      await profileApi.save(
        profileInput(own, {
          firstName: firstName.trim(),
          gender,
          bio: bio.trim(),
          city: city.trim(),
          interests,
        }),
      );
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      toast.show({ message: "Profil mis à jour.", kind: "success" });
      navigation.goBack();
    } catch (e) {
      setFormError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const birth = splitIsoDate(own.birthDate);
  return (
    <Screen scroll testID="edit-profile-screen">
      <TextField
        testID="edit-name"
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        maxLength={40}
      />
      <TextField
        label="Date de naissance"
        value={`${birth.day}/${birth.month}/${birth.year}`}
        editable={false}
        hint="Non modifiable."
      />
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" muted>
          Je suis
        </Text>
        <GenderPicker value={gender} onChange={setGender} testIDPrefix="edit-gender" />
      </View>
      <TextField
        testID="edit-bio"
        label="Bio"
        hint={`${bio.length}/500`}
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
      />
      <TextField
        testID="edit-city"
        label="Ville"
        value={city}
        onChangeText={setCity}
        maxLength={80}
      />
      <InterestPicker value={interests} onChange={setInterests} />
      <FormError testID="edit-error" message={formError} />
      <Button testID="edit-save" title="Enregistrer" onPress={save} loading={busy} />
    </Screen>
  );
}

export function ManagePhotosScreen() {
  const { data: own, error, isLoading, refetch } = useOwnProfile();
  if (isLoading) return <LoadingState />;
  if (error || !own) return <ErrorState error={error} onRetry={() => void refetch()} />;
  return (
    <Screen scroll testID="manage-photos-screen">
      <Text muted>
        Touchez une photo pour la définir comme principale, la déplacer, la remplacer ou la
        supprimer. Maximum 6 photos.
      </Text>
      <PhotoManager photos={own.photos} />
    </Screen>
  );
}

export function PreferencesScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const toast = useToast();
  const { data: own, error, isLoading, refetch } = useOwnProfile();
  const [prefs, setPrefs] = useState<Preferences>(DEFAULT_PREFERENCES);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (own?.preferences) setPrefs(own.preferences);
  }, [own]);

  if (isLoading) return <LoadingState />;
  if (error || !own) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const save = async () => {
    if (prefs.interestedIn.length === 0) {
      setFormError("Choisissez au moins un genre à rencontrer.");
      return;
    }
    setBusy(true);
    setFormError(null);
    try {
      await profileApi.savePreferences(prefs);
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      toast.show({ message: "Préférences enregistrées.", kind: "success" });
      navigation.goBack();
    } catch (e) {
      setFormError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const updateLocation = async () => {
    const outcome = await shareApproximateLocation();
    if (outcome === "saved")
      toast.show({ message: "Position approximative mise à jour.", kind: "success" });
    else
      toast.show({
        message:
          outcome === "denied"
            ? "Autorisez la localisation dans les réglages."
            : "Position indisponible.",
        kind: "error",
      });
    await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
  };

  const removeLocation = async () => {
    await profileApi.clearLocation();
    toast.show({ message: "Position supprimée.", kind: "success" });
    await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
  };

  return (
    <Screen scroll testID="preferences-screen">
      <PreferencesForm value={prefs} onChange={setPrefs} />
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" muted>
          Position approximative : {own.hasLocation ? "partagée" : "non partagée"}
        </Text>
        <Button
          testID="pref-location-update"
          title={own.hasLocation ? "Actualiser ma position" : "Partager ma position approximative"}
          variant="secondary"
          onPress={updateLocation}
        />
        {own.hasLocation ? (
          <Button
            testID="pref-location-clear"
            title="Supprimer ma position"
            variant="ghost"
            onPress={removeLocation}
          />
        ) : null}
      </View>
      <FormError testID="pref-error" message={formError} />
      <Button testID="pref-save" title="Enregistrer" onPress={save} loading={busy} />
    </Screen>
  );
}

export function PrivacyScreen() {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const toast = useToast();
  const { data: own, error, isLoading, refetch } = useOwnProfile();

  const update = useMutation({
    mutationFn: (patch: { discoverable?: boolean; showDistance?: boolean }) => {
      if (!own) throw new Error("Profil indisponible.");
      return profileApi.save(profileInput(own, patch));
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile }),
    onError: (e) => toast.show({ message: errorMessage(e), kind: "error" }),
  });

  if (isLoading) return <LoadingState />;
  if (error || !own) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const row = (
    label: string,
    hint: string,
    value: boolean,
    onChange: (v: boolean) => void,
    testID: string,
  ) => (
    <View style={[styles.switchRow, { borderBottomColor: colors.border }]}>
      <View style={styles.flex}>
        <Text variant="heading">{label}</Text>
        <Text muted>{hint}</Text>
      </View>
      <Switch
        testID={testID}
        accessibilityLabel={label}
        value={value}
        onValueChange={onChange}
        trackColor={{ true: colors.primary }}
      />
    </View>
  );

  return (
    <Screen scroll testID="privacy-screen">
      {row(
        "Profil visible",
        "Si vous le désactivez, vous n'apparaissez plus dans la découverte. Vos matchs actuels restent actifs.",
        own.discoverable,
        (v) => update.mutate({ discoverable: v }),
        "privacy-discoverable",
      )}
      {row(
        "Afficher ma distance",
        "Les autres voient une distance approximative (par tranche de 5 km), jamais votre position exacte.",
        own.showDistance,
        (v) => update.mutate({ showDistance: v }),
        "privacy-distance",
      )}
    </Screen>
  );
}

export function BlockedUsersScreen() {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const toast = useToast();
  const { data, error, isLoading, refetch } = useQuery({
    queryKey: queryKeys.blocked,
    queryFn: safetyApi.blocked,
  });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.blocked }),
    onError: (e) => toast.show({ message: errorMessage(e), kind: "error" }),
  });

  if (isLoading) return <LoadingState />;
  if (error) return <ErrorState error={error} onRetry={() => void refetch()} />;
  if (!data || data.length === 0) {
    return (
      <EmptyState
        testID="blocked-empty"
        icon="shield-checkmark-outline"
        title="Aucune personne bloquée"
      />
    );
  }
  return (
    <Screen padded={false} edges={["bottom"]} testID="blocked-screen">
      <FlatList
        data={data}
        keyExtractor={(b) => b.userId}
        renderItem={({ item }) => (
          <View style={[styles.switchRow, { borderBottomColor: colors.border }]}>
            <Text variant="heading" style={styles.flex}>
              {item.firstName || "Ancien profil"}
            </Text>
            <Button
              title="Débloquer"
              variant="secondary"
              onPress={() => unblock.mutate(item.userId)}
            />
          </View>
        )}
      />
    </Screen>
  );
}

export function DeleteAccountScreen() {
  const { deleteAccount } = useAuth();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const submit = async () => {
    if (!password) {
      setFormError("Entrez votre mot de passe pour confirmer.");
      return;
    }
    setBusy(true);
    setFormError(null);
    try {
      await deleteAccount(password);
    } catch (e) {
      setFormError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="delete-account-screen">
      <Text variant="title">Supprimer mon compte</Text>
      <Text>
        Cette action est définitive. Votre profil, vos photos, vos matchs, vos conversations et vos
        notifications sont supprimés immédiatement.
      </Text>
      <TextField
        testID="delete-password"
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
      />
      <FormError testID="delete-error" message={formError} />
      <Button
        testID="delete-confirm"
        title="Supprimer définitivement mon compte"
        variant="danger"
        onPress={submit}
        loading={busy}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, gap: 2 },
  switchRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.lg,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderRadius: radius.sm,
  },
});
