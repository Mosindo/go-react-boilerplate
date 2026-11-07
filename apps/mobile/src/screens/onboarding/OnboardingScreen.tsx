import React, { useState } from "react";
import { View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AppText, Banner, Button, Loading, Row, Screen } from "../../components/ui";
import PhotosEditor from "../../components/PhotosEditor";
import {
  AboutSection,
  IdentitySection,
  PreferencesSection,
  draftFrom,
  toInput,
  validateDraft,
  type Draft,
  type Section,
} from "../../components/ProfileSections";
import { profileApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { useAuth } from "../../auth/AuthContext";
import { spacing, useTheme } from "../../theme/theme";
import type { MyProfile } from "../../api/types";

const STEPS: { title: string; subtitle: string; sections: Section[] }[] = [
  {
    title: "Faisons connaissance",
    subtitle: "Ces informations composent votre profil.",
    sections: ["identity"],
  },
  {
    title: "Parlez-nous de vous",
    subtitle: "Une courte bio et quelques passions suffisent.",
    sections: ["about"],
  },
  { title: "Vos préférences", subtitle: "Modifiables à tout moment.", sections: ["preferences"] },
  {
    title: "Vos plus belles photos",
    subtitle: "Au moins une photo est nécessaire pour découvrir des profils.",
    sections: [],
  },
];

export default function OnboardingScreen({
  existing,
  onFinish,
}: {
  existing: MyProfile | null;
  onFinish: () => void;
}) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const qc = useQueryClient();
  const [step, setStep] = useState(existing ? 3 : 0);
  const [draft, setDraft] = useState<Draft>(() => draftFrom(existing));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () => profileApi.save(toInput(draft)),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["profile"] });
      setStep(3);
    },
    onError: (e) => setError(errorMessage(e)),
  });

  // Photos step reads the live profile so uploads appear immediately.
  const profile = useQuery({
    queryKey: ["profile"],
    queryFn: profileApi.mine,
    enabled: step === 3,
    retry: false,
  });

  const set = (patch: Partial<Draft>) => {
    setDraft((d) => ({ ...d, ...patch }));
    setErrors({});
  };

  const current = STEPS[step]!;
  const next = () => {
    setError(null);
    const found = validateDraft(draft, current.sections);
    if (Object.keys(found).length > 0) return setErrors(found);
    if (step === 2) save.mutate();
    else setStep(step + 1);
  };

  const photos = profile.data?.photos ?? [];

  return (
    <Screen scroll>
      <Row style={{ gap: spacing.xs }}>
        {STEPS.map((_, i) => (
          <View
            key={i}
            style={{
              flex: 1,
              height: 4,
              borderRadius: 2,
              backgroundColor: i <= step ? colors.primary : colors.border,
            }}
          />
        ))}
      </Row>
      <View style={{ gap: spacing.xs }}>
        <AppText variant="title" testID="onboarding-title">
          {current.title}
        </AppText>
        <AppText variant="muted">{current.subtitle}</AppText>
      </View>
      {error ? <Banner message={error} /> : null}
      {step === 0 ? <IdentitySection draft={draft} set={set} errors={errors} /> : null}
      {step === 1 ? <AboutSection draft={draft} set={set} errors={errors} /> : null}
      {step === 2 ? <PreferencesSection draft={draft} set={set} errors={errors} /> : null}
      {step === 3 ? profile.isLoading ? <Loading /> : <PhotosEditor photos={photos} /> : null}
      {step < 3 ? (
        <Button
          testID="onboarding-next"
          title={step === 2 ? "Enregistrer et continuer" : "Continuer"}
          onPress={next}
          loading={save.isPending}
        />
      ) : (
        <Button
          testID="onboarding-finish"
          title="Découvrir des profils"
          disabled={photos.length === 0}
          onPress={() => {
            void qc.invalidateQueries({ queryKey: ["profile"] });
            onFinish();
          }}
        />
      )}
      <Row style={{ justifyContent: "space-between" }}>
        {step > 0 && !(existing && step === 3) ? (
          <Button title="Retour" variant="ghost" onPress={() => setStep(step - 1)} />
        ) : (
          <View />
        )}
        <Button title="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </Row>
    </Screen>
  );
}
