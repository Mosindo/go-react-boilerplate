import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { describeError } from "../api/client";
import type { Preferences } from "../api/types";
import { PreferencesFields } from "../components/Pickers";
import { useMe, useSavePreferences } from "../hooks/data";
import type { MainStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Notice } from "../shared/ui";

export default function PreferencesScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "Preferences">) {
  const { data: me, isLoading } = useMe();
  const save = useSavePreferences();
  const [value, setValue] = useState<Preferences | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (isLoading) return <LoadingView fullScreen />;
  if (!me) return <ErrorView message="Impossible de charger vos préférences." />;
  const current = value ?? me.preferences;

  const submit = async () => {
    setError(null);
    try {
      await save.mutateAsync(current);
      showToast("Préférences enregistrées.", "success");
      navigation.goBack();
    } catch (e) {
      setError(describeError(e));
    }
  };

  return (
    <ScreenContainer
      footer={
        <View style={{ width: "100%", maxWidth: 520 }}>
          <Button fullWidth loading={save.isPending} onPress={submit} size="lg">
            Enregistrer
          </Button>
        </View>
      }
    >
      {error ? <Notice message={error} tone="error" /> : null}
      <PreferencesFields onChange={setValue} value={current} />
    </ScreenContainer>
  );
}
