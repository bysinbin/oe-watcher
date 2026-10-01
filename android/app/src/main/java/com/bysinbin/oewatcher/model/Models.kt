package com.bysinbin.oewatcher.model

import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.UUID

data class PersonItem(
    val name: String,
    val tags: List<String> = emptyList(),
    val topics: List<String> = emptyList()
)

data class MediaTrackInfo(
    val app: String,
    val packageName: String,
    val title: String,
    val artist: String,
    val album: String = "",
    val isPlaying: Boolean = true,
    val timestamp: Long = System.currentTimeMillis()
)

data class MatchResult(
    val matched: Boolean,
    val personName: String,
    val category: String,
    val matchedField: String
)

data class SkipEvent(
    val id: String = UUID.randomUUID().toString(),
    val app: String,
    val artist: String,
    val title: String,
    val matchedPerson: String,
    val timestamp: String = SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date()),
    val success: Boolean = true
)

data class AliasItem(
    val alias: String,
    val target: String
)

data class ExcludeItem(
    val pattern: String
)
