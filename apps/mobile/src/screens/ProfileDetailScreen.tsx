import React, { useRef, useState } from "react";
import {
  FlatList,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
  useWindowDimensions
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/errors";
import { discoverApi, profileApi, safetyApi } from "../api/services";
import { AuthImage } from "../components/AuthImage";
import { ConfirmSheet, ReportSheet } from "../components/SafetySheets";
import { Sheet } from "../components/Sheet";
import { useToast } from "../components/Toast";
import { Button, Chip, ErrorState, IconButton, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { emitProfileDecided } from "../lib/deckBus";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";

export function ProfileDetailScreen({
  route,
  navigation
}: NativeStackScreenProps<RootStackParamList, "ProfileDetail">) {
  const { userId, canSwipe } = route.params;
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { width } = useWindowDimensions();
  const pageWidth = Math.min(width, 560);
  const [page, setPage] = useState(0);
  const [menu, setMenu] = useState(false);
  const [report, setReport] = useState(false);
  const [confirmBlock, setConfirmBlock] = useState(false);
  const list = useRef<FlatList>(null);

  const query = useQuery({
    queryKey: ["profile", userId],
    queryFn: () => profileApi.view(userId),
    retry: false
  });

  const decide = async (action: "like" | "pass") => {
    try {
      const result = await discoverApi.swipe(userId, action);
      emitProfileDecided(userId);
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      if (result.matched && !result.duplicate && result.conversationId) {
        navigation.replace("Chat", {
          conversationId: result.conversationId,
          name: query.data?.firstName
        });
      } else {
        navigation.goBack();
      }
    } catch (error) {
      toast.show(errorMessage(error, fr.discover.swipeFailed), "error");
    }
  };

  const block = async () => {
    try {
      await safetyApi.block(userId);
      toast.show(fr.safety.blocked, "success");
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["blocks"] });
      setConfirmBlock(false);
      navigation.goBack();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  if (query.isLoading)
    return (
      <Screen edges={[]}>
        <Loader />
      </Screen>
    );
  if (query.isError || !query.data) {
    return (
      <Screen edges={[]}>
        <ErrorState title={fr.profileView.gone} />
      </Screen>
    );
  }
  const p = query.data;

  return (
    <Screen edges={["bottom"]}>
      <ScrollView testID="profile-detail">
        <View style={{ alignSelf: "center", width: pageWidth }}>
          <FlatList
            ref={list}
            horizontal
            pagingEnabled
            showsHorizontalScrollIndicator={false}
            data={p.photos}
            keyExtractor={(ph) => ph.id}
            onMomentumScrollEnd={(e) =>
              setPage(Math.round(e.nativeEvent.contentOffset.x / pageWidth))
            }
            renderItem={({ item, index }) => (
              <View style={{ width: pageWidth, aspectRatio: 3 / 4 }}>
                <AuthImage
                  photoId={item.id}
                  style={StyleSheet.absoluteFill as object}
                  accessibilityLabel={`Photo ${index + 1} de ${p.firstName}`}
                />
              </View>
            )}
          />
          {p.photos.length > 1 ? (
            <View
              style={{
                position: "absolute",
                top: 10,
                left: 12,
                right: 12,
                flexDirection: "row",
                gap: 4
              }}
              pointerEvents="none"
            >
              {p.photos.map((ph, i) => (
                <View
                  key={ph.id}
                  style={{
                    flex: 1,
                    height: 3,
                    borderRadius: 2,
                    backgroundColor: i === page ? "#FFF" : "rgba(255,255,255,0.45)"
                  }}
                />
              ))}
            </View>
          ) : null}
        </View>
        <View
          style={{
            padding: spacing.xl,
            gap: spacing.lg,
            maxWidth: 560,
            width: "100%",
            alignSelf: "center"
          }}
        >
          <View
            style={{
              flexDirection: "row",
              alignItems: "flex-start",
              justifyContent: "space-between"
            }}
          >
            <View style={{ flex: 1 }}>
              <Text variant="title" testID="detail-name">
                {p.firstName}, {p.age}
              </Text>
              <Text muted>
                {[p.city, p.distanceKm !== null ? fr.discover.kmAway(p.distanceKm) : null]
                  .filter(Boolean)
                  .join(" · ")}
              </Text>
            </View>
            <Pressable
              testID="detail-menu"
              accessibilityRole="button"
              accessibilityLabel={fr.messages.options}
              onPress={() => setMenu(true)}
              hitSlop={12}
            >
              <Text variant="heading" color={colors.primary}>
                ⋯
              </Text>
            </Pressable>
          </View>
          {p.bio ? (
            <View style={{ gap: spacing.xs }}>
              <Text variant="label" muted>
                {fr.profileView.about}
              </Text>
              <Text>{p.bio}</Text>
            </View>
          ) : null}
          {p.interests.length > 0 ? (
            <View style={{ gap: spacing.sm }}>
              <Text variant="label" muted>
                {fr.profileView.interests}
              </Text>
              <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
                {p.interests.map((i) => (
                  <Chip key={i.slug} label={i.label} />
                ))}
              </View>
            </View>
          ) : null}
        </View>
      </ScrollView>
      {canSwipe ? (
        <View
          style={{
            flexDirection: "row",
            justifyContent: "center",
            gap: spacing.xl,
            paddingVertical: spacing.md
          }}
        >
          <IconButton
            testID="detail-pass"
            label={fr.discover.pass}
            glyph="✕"
            onPress={() => void decide("pass")}
          />
          <IconButton
            testID="detail-like"
            label={fr.discover.like}
            glyph="♥"
            tone="primary"
            onPress={() => void decide("like")}
          />
        </View>
      ) : null}

      <Sheet visible={menu} onClose={() => setMenu(false)} title={p.firstName}>
        <Button
          testID="detail-report"
          variant="secondary"
          title={fr.safety.report}
          onPress={() => {
            setMenu(false);
            setReport(true);
          }}
        />
        <Button
          testID="detail-block"
          variant="secondary"
          title={fr.safety.block}
          onPress={() => {
            setMenu(false);
            setConfirmBlock(true);
          }}
        />
        <Button variant="ghost" title={fr.common.close} onPress={() => setMenu(false)} />
      </Sheet>
      <ReportSheet
        visible={report}
        userId={userId}
        onClose={() => setReport(false)}
        onDone={(blocked) => {
          setReport(false);
          if (blocked) navigation.goBack();
        }}
      />
      <ConfirmSheet
        visible={confirmBlock}
        title={fr.safety.block}
        message={fr.safety.blockConfirm(p.firstName)}
        confirmLabel={fr.safety.block}
        danger
        onConfirm={() => void block()}
        onClose={() => setConfirmBlock(false)}
      />
    </Screen>
  );
}
