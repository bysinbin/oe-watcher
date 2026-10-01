package com.bysinbin.oewatcher.engine

import android.content.Context
import android.util.Log
import com.bysinbin.oewatcher.model.MatchResult
import com.bysinbin.oewatcher.model.PersonItem
import org.json.JSONObject
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.Locale
import java.util.regex.Pattern

object BlacklistMatcher {
    private const val TAG = "BlacklistMatcher"

    data class CompiledMatcher(
        val displayName: String,
        val candidate: String,
        val category: String,
        val pattern: Pattern
    )

    private val matchers = mutableListOf<CompiledMatcher>()
    private val basePersons = mutableListOf<PersonItem>()
    private val customPersons = mutableListOf<PersonItem>()
    private val baseAliases = mutableMapOf<String, String>()
    private val customAliases = mutableMapOf<String, String>()
    private val baseExcludes = mutableSetOf<String>()
    private val customExcludes = mutableSetOf<String>()

    val totalCount: Int
        get() = basePersons.size + customPersons.size

    fun foldTurkish(str: String?): String {
        if (str.isNullOrBlank()) return ""
        return str
            .replace('İ', 'i')
            .replace('I', 'ı')
            .replace('Ç', 'ç')
            .replace('Ğ', 'ğ')
            .replace('Ö', 'ö')
            .replace('Ş', 'ş')
            .replace('Ü', 'ü')
            .lowercase(Locale.forLanguageTag("tr-TR"))
            .replace('ç', 'c')
            .replace('ğ', 'g')
            .replace('ı', 'i')
            .replace('ö', 'o')
            .replace('ş', 's')
            .replace('ü', 'u')
            .trim()
    }

    private fun extractCandidateNames(rawName: String): List<String> {
        val list = mutableSetOf<String>()
        val clean = rawName.trim()
        if (clean.isBlank()) return emptyList()

        val parenMatch = Regex("^(.*?)\\s*\\((.*?)\\)\\s*$").find(clean)
        if (parenMatch != null) {
            val primary = parenMatch.groupValues[1].trim()
            val secondary = parenMatch.groupValues[2].trim()
            if (primary.length >= 3) list.add(primary)
            if (secondary.length >= 3) list.add(secondary)
        }
        if (clean.length >= 3) {
            list.add(clean)
        }
        return list.toList()
    }

    @Synchronized
    fun loadFromAssets(context: Context) {
        try {
            val inputStream = context.assets.open("default_persons.json")
            val reader = BufferedReader(InputStreamReader(inputStream))
            val sb = java.lang.StringBuilder()
            var line: String?
            while (reader.readLine().also { line = it } != null) {
                sb.append(line)
            }
            reader.close()

            val root = JSONObject(sb.toString())
            parseJsonData(root)
            Log.d(TAG, "Assets verisi başarıyla yüklendi: ${basePersons.size} kişi, ${matchers.size} kural.")
        } catch (e: Exception) {
            Log.e(TAG, "Asset yükleme hatası: ${e.message}", e)
        }
    }

    @Synchronized
    fun parseJsonData(root: JSONObject) {
        basePersons.clear()
        baseAliases.clear()
        baseExcludes.clear()

        // Exclude list
        if (root.has("exclude_list")) {
            val excludes = root.getJSONArray("exclude_list")
            for (i in 0 until excludes.length()) {
                val ex = excludes.optString(i)
                if (ex.isNotBlank()) baseExcludes.add(ex.trim())
            }
        }

        // Aliases
        if (root.has("aliases")) {
            val aliasObj = root.getJSONObject("aliases")
            val keys = aliasObj.keys()
            while (keys.hasNext()) {
                val k = keys.next()
                baseAliases[k] = aliasObj.getString(k)
            }
        }

        // Persons
        val personsArray = root.optJSONArray("persons")
        if (personsArray != null) {
            for (i in 0 until personsArray.length()) {
                val obj = personsArray.getJSONObject(i)
                val name = obj.optString("name", "")
                if (name.isBlank()) continue

                val tags = mutableListOf<String>()
                val tagsArr = obj.optJSONArray("tags")
                if (tagsArr != null) {
                    for (j in 0 until tagsArr.length()) tags.add(tagsArr.getString(j))
                }

                val topics = mutableListOf<String>()
                val topicsArr = obj.optJSONArray("topics")
                if (topicsArr != null) {
                    for (j in 0 until topicsArr.length()) topics.add(topicsArr.getString(j))
                }

                basePersons.add(PersonItem(name, tags, topics))
            }
        }

        rebuildMatchers()
    }

    @Synchronized
    fun getAllPersons(): List<PersonItem> = (customPersons + basePersons).toList()

    @Synchronized
    fun getBasePersons(): List<PersonItem> = basePersons.toList()

    @Synchronized
    fun getCustomPersons(): List<PersonItem> = customPersons.toList()

    @Synchronized
    fun getAllAliases(): Map<String, String> = (baseAliases + customAliases).toMap()

    @Synchronized
    fun getCustomAliases(): Map<String, String> = customAliases.toMap()

    @Synchronized
    fun getAllExcludes(): List<String> = (baseExcludes + customExcludes).toList()

    @Synchronized
    fun getCustomExcludes(): List<String> = customExcludes.toList()

    @Synchronized
    fun setCustomPersons(list: List<PersonItem>) {
        customPersons.clear()
        customPersons.addAll(list)
        rebuildMatchers()
    }

    @Synchronized
    fun addCustomPerson(name: String, tags: List<String>, topics: List<String> = emptyList()) {
        val person = PersonItem(name.trim(), tags, topics)
        customPersons.removeAll { it.name.equals(name.trim(), ignoreCase = true) }
        customPersons.add(0, person)
        rebuildMatchers()
    }

    @Synchronized
    fun removeCustomPerson(name: String) {
        customPersons.removeAll { it.name.equals(name.trim(), ignoreCase = true) }
        rebuildMatchers()
    }

    @Synchronized
    fun setCustomAliases(map: Map<String, String>) {
        customAliases.clear()
        customAliases.putAll(map)
        rebuildMatchers()
    }

    @Synchronized
    fun addCustomAlias(alias: String, target: String) {
        customAliases[alias.trim()] = target.trim()
        rebuildMatchers()
    }

    @Synchronized
    fun removeCustomAlias(alias: String) {
        customAliases.remove(alias.trim())
        rebuildMatchers()
    }

    @Synchronized
    fun setCustomExcludes(list: List<String>) {
        customExcludes.clear()
        customExcludes.addAll(list.map { it.trim() }.filter { it.isNotBlank() })
        rebuildMatchers()
    }

    @Synchronized
    fun addCustomExclude(pattern: String) {
        val p = pattern.trim()
        if (p.isNotBlank()) {
            customExcludes.add(p)
            rebuildMatchers()
        }
    }

    @Synchronized
    fun removeCustomExclude(pattern: String) {
        customExcludes.remove(pattern.trim())
        rebuildMatchers()
    }

    @Synchronized
    fun isExcluded(target: String?): Boolean {
        if (target.isNullOrBlank()) return false
        val foldedTarget = foldTurkish(target)
        val allEx = (baseExcludes + customExcludes)
        for (ex in allEx) {
            val foldedEx = foldTurkish(ex)
            if (foldedEx.isNotBlank() && foldedTarget.contains(foldedEx)) {
                return true
            }
        }
        return false
    }

    @Synchronized
    private fun rebuildMatchers() {
        matchers.clear()
        val allExcludesSet = (baseExcludes + customExcludes).map { foldTurkish(it) }.toSet()

        // Index all persons (custom first, then base)
        val allPersons = customPersons + basePersons
        for (person in allPersons) {
            val candidates = extractCandidateNames(person.name)
            for (cand in candidates) {
                val folded = foldTurkish(cand)
                if (folded.length < 3 || allExcludesSet.contains(folded)) continue
                val patternStr = "(?:^|[^a-z0-9])${Pattern.quote(folded)}(?:$|[^a-z0-9])"
                val pattern = Pattern.compile(patternStr, Pattern.CASE_INSENSITIVE)
                matchers.add(
                    CompiledMatcher(
                        displayName = person.name,
                        candidate = cand,
                        category = person.tags.firstOrNull() ?: "Kara Liste",
                        pattern = pattern
                    )
                )
            }
        }

        // Index all aliases (custom + base)
        val allAliases = baseAliases + customAliases
        for ((alias, target) in allAliases) {
            val folded = foldTurkish(alias)
            if (folded.length < 3 || allExcludesSet.contains(folded)) continue
            val patternStr = "(?:^|[^a-z0-9])${Pattern.quote(folded)}(?:$|[^a-z0-9])"
            val pattern = Pattern.compile(patternStr, Pattern.CASE_INSENSITIVE)
            matchers.add(
                CompiledMatcher(
                    displayName = "$target (Alias: $alias)",
                    candidate = alias,
                    category = "Özel Alias",
                    pattern = pattern
                )
            )
        }
    }

    @Synchronized
    fun match(artist: String?, title: String?): MatchResult? {
        val cleanArtist = artist?.trim() ?: ""
        val cleanTitle = title?.trim() ?: ""

        if (cleanArtist.isBlank() && cleanTitle.isBlank()) return null

        // 1. Whitelist / Exclude check
        if (isExcluded(cleanArtist) || isExcluded(cleanTitle)) {
            Log.d(TAG, "Muaf / Whitelist eşleşti, atlama yapılmayacak: '$cleanArtist' - '$cleanTitle'")
            return null
        }

        val foldedArtist = foldTurkish(cleanArtist)
        val foldedTitle = foldTurkish(cleanTitle)

        // 2. Check artist matches
        if (foldedArtist.isNotBlank()) {
            for (m in matchers) {
                if (m.pattern.matcher(foldedArtist).find()) {
                    return MatchResult(
                        matched = true,
                        personName = m.displayName,
                        category = m.category,
                        matchedField = cleanArtist
                    )
                }
            }
        }

        // 3. Check title matches (useful for YouTube / compilations / covers)
        if (foldedTitle.isNotBlank()) {
            for (m in matchers) {
                if (m.pattern.matcher(foldedTitle).find()) {
                    return MatchResult(
                        matched = true,
                        personName = m.displayName,
                        category = m.category,
                        matchedField = cleanTitle
                    )
                }
            }
        }

        return null
    }
}
