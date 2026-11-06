import React, { useEffect, useState } from "react";
import { Pressable, ScrollView, Switch, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/errors";
import { authApi, photoApi, profileApi, safetyApi } from "../api/services";
import type { Gender, MyProfile } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { Avatar } from "../components/Avatar";
import { PhotoManager } from "../components/PhotoGrid";
import { BasicsFields, basicsErrors, GenderPicker, InterestPicker, LocationBlock, type BasicsDraft } from "../components/ProfileForm";
import { ConfirmSheet } from "../components/SafetySheets";
import { useToast } from "../components/Toast";
import { Button, Card, Chip, Divider, ErrorState, Input, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { locateDevice } from "../lib/device";
import { displayBirthDate, parseBirthDate } from "../lib/format";
import { isValidPassword } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";
import { Stepper } from "./OnboardingScreen";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function useMyProfile() {
  return useQuery({ queryKey: ["me", "profile"], queryFn: profileApi.mine });
}

function Page({ children }: { children: React.ReactNode }) {
  return (
    <Screen edges={["bottom", "left", "right"]}>
      <ScrollView contentContainerStyle={{ padding: spacing.xl, gap: spacing.lg }} keyboardShouldPersistTaps="handled">
        <View style={{ maxWidth: 520, width: "100%", alignSelf: "center", gap: spacing.lg }}>{children}</View>
      </ScrollView>
    </Screen>
  );
}

function MenuRow({ label, onPress, testID, danger }: { label: string; onPress: () => void; testID?: string; danger?: boolean }) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => ({
        flexDirection: "row",
        justifyContent: "space-between",
        alignItems: "center",
        minHeight: 52,
        paddingHorizontal: spacing.lg,
        backgroundColor: pressed ? colors.surfaceAlt : "transparent"
      })}
    >
      <Text color={danger ? colors.danger : colors.text}>{label}</Text>
      <Text muted>›</Text>
    </Pressable>
  );
}

export function ProfileScreen() {
  const navigation = useNavigation<Nav>();
  const auth = useAuth();
  const { colors } = useTheme();
  const query = useMyProfile();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const remove = async () => {
    setBusy(true);
    setDeleteError(null);
    try {
      await auth.deleteAccount(deletePassword);
    } catch (error) {
      setDeleteError(errorMessage(error));
      setBusy(false);
    }
  };

  if (query.isLoading) return <Screen><Loader /></Screen>;
  if (query.isError || !query.data?.profile) return <Screen><ErrorState onRetry={() => void query.refetch()} /></Screen>;
  const { profile, photos, interests } = query.data;

  return (
    <Screen>
      <ScrollView contentContainerStyle={{ paddingBottom: spacing.xxl }}>
        <View style={{ maxWidth: 520, width: "100%", alignSelf: "center" }}>
          <View style={{ alignItems: "center", padding: spacing.xl, gap: spacing.sm }}>
            <Avatar name={profile.firstName} photoId={photos[0]?.id} size={112} />
            <Text variant="title" testID="me-name">
              {profile.firstName}, {profile.age}
            </Text>
            <Text muted>{profile.city}</Text>
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 6, justifyContent: "center" }}>
              {interests.map((i) => (
                <Chip key={i.slug} label={i.label} />
              ))}
            </View>
            {!profile.discoverable ? (
              <Text variant="caption" color={colors.danger} testID="paused-notice">
                {fr.profile.discoverableHint}
              </Text>
            ) : null}
          </View>
          <Card style={{ marginHorizontal: spacing.lg, padding: 0, overflow: "hidden" }}>
            <MenuRow testID="menu-edit" label={fr.profile.edit} onPress={() => navigation.navigate("EditProfile")} />
            <Divider />
            <MenuRow testID="menu-photos" label={fr.profile.photos} onPress={() => navigation.navigate("Photos")} />
            <Divider />
            <MenuRow testID="menu-prefs" label={fr.profile.preferences} onPress={() => navigation.navigate("Preferences")} />
            <Divider />
            <MenuRow testID="menu-privacy" label={fr.profile.privacy} onPress={() => navigation.navigate("Privacy")} />
            <Divider />
            <MenuRow testID="menu-password" label={fr.profile.password} onPress={() => navigation.navigate("ChangePassword")} />
          </Card>
          <View style={{ padding: spacing.lg, gap: spacing.sm }}>
            <Button testID="logout" variant="secondary" title={fr.profile.logout} onPress={() => void auth.logout()} />
            <Button testID="delete-account" variant="ghost" title={fr.profile.deleteAccount} onPress={() => setConfirmDelete(true)} />
          </View>
        </View>
      </ScrollView>
      <ConfirmSheet
        visible={confirmDelete}
        title={fr.profile.deleteAccount}
        message={fr.profile.deleteWarning}
        confirmLabel={fr.profile.deleteAccount}
        danger
        busy={busy}
        onConfirm={() => void remove()}
        onClose={() => {
          setConfirmDelete(false);
          setDeletePassword("");
          setDeleteError(null);
        }}
      >
        <Input testID="delete-password" label={fr.auth.password} value={deletePassword} onChangeText={setDeletePassword} secureTextEntry autoCapitalize="none" error={deleteError} />
      </ConfirmSheet>
    </Screen>
  );
}

export function EditProfileScreen() {
  const navigation = useNavigation<Nav>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useMyProfile();
  const interestsQuery = useQuery({ queryKey: ["interests"], queryFn: profileApi.interests, staleTime: Infinity });
  const [basics, setBasics] = useState<BasicsDraft>({ firstName: "", birthDate: "", gender: null });
  const [bio, setBio] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [place, setPlace] = useState<{ latitude: number; longitude: number; city: string } | null>(null);
  const [city, setCity] = useState("");
  const [locating, setLocating] = useState(false);
  const [denied, setDenied] = useState(false);
  const [show, setShow] = useState(false);
  const [loaded, setLoaded] = useState(false);

  const data = query.data;
  useEffect(() => {
    if (!data?.profile || loaded) return;
    setBasics({ firstName: data.profile.firstName, birthDate: displayBirthDate(data.profile.birthDate), gender: data.profile.gender });
    setBio(data.profile.bio);
    setCity(data.profile.city);
    setInterests(data.interests.map((i) => i.slug));
    setLoaded(true);
  }, [data, loaded]);

  const save = useMutation({
    mutationFn: async () => {
      const birthDate = parseBirthDate(basics.birthDate);
      if (!birthDate || !basics.gender || !place) throw new Error("incomplete");
      return profileApi.save({
        firstName: basics.firstName.trim(),
        birthDate,
        gender: basics.gender,
        bio: bio.trim(),
        city: place.city,
        latitude: place.latitude,
        longitude: place.longitude,
        interests
      });
    },
    onSuccess: (profile) => {
      queryClient.setQueryData<MyProfile>(["me", "profile"], profile);
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      toast.show(fr.profile.saved, "success");
      navigation.goBack();
    },
    onError: (error) => toast.show(errorMessage(error), "error")
  });

  if (!loaded) return <Screen edges={[]}><Loader /></Screen>;

  const submit = () => {
    setShow(true);
    if (Object.keys(basicsErrors(basics)).length > 0) return;
    if (!place) return toast.show(fr.onboarding.locationDenied, "error");
    save.mutate();
  };

  return (
    <Page>
      <BasicsFields draft={basics} onChange={setBasics} showErrors={show} />
      <Input testID="edit-bio" label={fr.onboarding.bio} hint={fr.onboarding.bioHint} value={bio} onChangeText={setBio} multiline maxLength={500} />
      <Text variant="label">{fr.onboarding.interests}</Text>
      <InterestPicker all={interestsQuery.data ?? []} selected={interests} onChange={setInterests} />
      <Text variant="label">{fr.onboarding.locationTitle}</Text>
      <Text muted>{city ? `Ville actuelle : ${city}. Mettez à jour votre position pour enregistrer les modifications.` : ""}</Text>
      <LocationBlock
        city={place?.city ?? ""}
        hasLocation={!!place}
        busy={locating}
        denied={denied}
        onLocate={() => {
          setLocating(true);
          setDenied(false);
          void locateDevice().then((r) => {
            setLocating(false);
            if (r) setPlace(r);
            else setDenied(true);
          });
        }}
        onPickCity={(c) => setPlace({ latitude: c.latitude, longitude: c.longitude, city: c.name })}
      />
      <Button testID="edit-save" title={save.isPending ? fr.common.saving : fr.common.save} loading={save.isPending} onPress={submit} />
    </Page>
  );
}

export function PhotosScreen() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["me", "photos"], queryFn: photoApi.list });
  if (query.isLoading) return <Screen edges={[]}><Loader /></Screen>;
  return (
    <Page>
      <Text muted>{fr.onboarding.photosIntro}</Text>
      <PhotoManager
        photos={query.data ?? []}
        onChanged={() => {
          void queryClient.invalidateQueries({ queryKey: ["me"] });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void query.refetch();
        }}
      />
    </Page>
  );
}

const SEEKING: Gender[] = ["woman", "man", "nonbinary"];

export function PreferencesScreen() {
  const navigation = useNavigation<Nav>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useMyProfile();
  const [interestedIn, setInterestedIn] = useState<Gender[]>([]);
  const [range, setRange] = useState<[number, number]>([18, 99]);
  const [distance, setDistance] = useState(50);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (!query.data || loaded) return;
    const p = query.data.preferences;
    setInterestedIn(p.interestedIn);
    setRange([p.minAge, p.maxAge]);
    setDistance(p.maxDistanceKm);
    setLoaded(true);
  }, [query.data, loaded]);

  const save = useMutation({
    mutationFn: () => profileApi.savePreferences({ interestedIn, minAge: range[0], maxAge: range[1], maxDistanceKm: distance }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["me"] });
      toast.show(fr.profile.saved, "success");
      navigation.goBack();
    },
    onError: (error) => toast.show(errorMessage(error), "error")
  });

  if (!loaded) return <Screen edges={[]}><Loader /></Screen>;
  return (
    <Page>
      <Text variant="label">{fr.onboarding.interestedIn}</Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {SEEKING.map((g) => (
          <Chip
            key={g}
            testID={`seek-${g}`}
            label={fr.onboarding.interestedOptions[g] ?? g}
            selected={interestedIn.includes(g)}
            onPress={() => setInterestedIn((cur) => (cur.includes(g) ? (cur.length > 1 ? cur.filter((x) => x !== g) : cur) : [...cur, g]))}
          />
        ))}
      </View>
      <Stepper
        label={`Âge minimum : ${range[0]} ans`}
        minusLabel="Âge minimum -"
        plusLabel="Âge minimum +"
        onMinus={() => setRange(([a, b]) => [Math.max(18, a - 1), b])}
        onPlus={() => setRange(([a, b]) => [Math.min(b, a + 1), b])}
      />
      <Stepper
        label={`Âge maximum : ${range[1]} ans`}
        minusLabel="Âge maximum -"
        plusLabel="Âge maximum +"
        onMinus={() => setRange(([a, b]) => [a, Math.max(a, b - 1)])}
        onPlus={() => setRange(([a, b]) => [a, Math.min(99, b + 1)])}
      />
      <Stepper
        label={`${fr.onboarding.maxDistance} : ${distance} km`}
        minusLabel="Distance -"
        plusLabel="Distance +"
        onMinus={() => setDistance((d) => Math.max(1, d <= 10 ? d - 1 : d - 5))}
        onPlus={() => setDistance((d) => Math.min(500, d < 10 ? d + 1 : d + 5))}
      />
      <Button testID="prefs-save" title={fr.common.save} loading={save.isPending} onPress={() => save.mutate()} />
    </Page>
  );
}

export function PrivacyScreen() {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const blocks = useQuery({ queryKey: ["blocks"], queryFn: safetyApi.blocks });

  const privacy = useMutation({
    mutationFn: (v: { discoverable: boolean; showDistance: boolean }) => profileApi.setPrivacy(v.discoverable, v.showDistance),
    onSuccess: (data) => queryClient.setQueryData<MyProfile>(["me", "profile"], data),
    onError: (error) => toast.show(errorMessage(error), "error")
  });
  const unblock = useMutation({
    mutationFn: (id: string) => safetyApi.unblock(id),
    onSuccess: () => void blocks.refetch(),
    onError: (error) => toast.show(errorMessage(error), "error")
  });

  const p = profile.data?.profile;
  if (!p) return <Screen edges={[]}><Loader /></Screen>;
  return (
    <Page>
      <Card style={{ gap: spacing.lg }}>
        <ToggleRow
          testID="toggle-discoverable"
          label={fr.profile.discoverable}
          hint={fr.profile.discoverableHint}
          value={p.discoverable}
          onChange={(v) => privacy.mutate({ discoverable: v, showDistance: p.showDistance })}
        />
        <ToggleRow
          testID="toggle-distance"
          label={fr.profile.showDistance}
          value={p.showDistance}
          onChange={(v) => privacy.mutate({ discoverable: p.discoverable, showDistance: v })}
        />
      </Card>
      <Text variant="heading">{fr.profile.blocked}</Text>
      {blocks.isLoading ? <Loader /> : null}
      {blocks.data && blocks.data.length === 0 ? <Text muted>{fr.profile.noBlocked}</Text> : null}
      {blocks.data?.map((b) => (
        <View key={b.userId} style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
          <Text>{b.firstName || "—"}</Text>
          <Button variant="secondary" title={fr.profile.unblock} onPress={() => unblock.mutate(b.userId)} />
        </View>
      ))}
      <Text variant="caption" color={colors.textMuted}>
        {" "}
      </Text>
    </Page>
  );
}

function ToggleRow({ label, hint, value, onChange, testID }: { label: string; hint?: string; value: boolean; onChange: (v: boolean) => void; testID?: string }) {
  const { colors } = useTheme();
  return (
    <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
      <View style={{ flex: 1, gap: 2 }}>
        <Text variant="label">{label}</Text>
        {hint ? (
          <Text variant="caption" muted>
            {hint}
          </Text>
        ) : null}
      </View>
      <Switch
        testID={testID}
        accessibilityLabel={label}
        value={value}
        onValueChange={onChange}
        trackColor={{ true: colors.primary, false: colors.border }}
      />
    </View>
  );
}

export function ChangePasswordScreen() {
  const auth = useAuth();
  const toast = useToast();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setError(null);
    if (!isValidPassword(next)) return setError(fr.auth.invalidPassword);
    setBusy(true);
    try {
      await authApi.changePassword(current, next);
      toast.show(fr.profile.passwordChanged, "success");
      // The server revoked every session: return to the login screen.
      await auth.forgetSession();
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Page>
      <Input testID="pw-current" label={fr.profile.currentPassword} value={current} onChangeText={setCurrent} secureTextEntry autoCapitalize="none" />
      <Input testID="pw-new" label={fr.auth.newPassword} value={next} onChangeText={setNext} secureTextEntry autoCapitalize="none" hint={fr.auth.passwordHint} error={error} />
      <Button testID="pw-submit" title={fr.common.save} loading={busy} onPress={() => void submit()} />
    </Page>
  );
}

export { GenderPicker };
