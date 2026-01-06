"""
X-Ray Demo: Competitor Selection Pipeline

Demonstrates a competitor discovery flow:
1. LLM Call generates search keywords from product info 
2. LLM Call retrieves candidate products - generating random Amazon ASINs similar to given keywords
3. LLM Call generates set of filters to apply to candidate products
4. Apply filters (price, rating, category) - non-llm step
5. LLM ranking of remaining candidates
6. Select best match

Requires: GROQ_API_KEY environment variable
"""

import os
import json
from typing import List, Dict, Any, Tuple

from groq import Groq

import xray_sdk as xray
from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode, SamplingConfig


# =============================================================================
# X-Ray Type Definitions
# =============================================================================

class Pipelines(XRayPipelineID):
    COMPETITOR_SELECTION = "competitor-selection"


class Steps(XRayStepType):
    KEYWORD_GEN = "keyword_generation"
    FILTER = "filter"
    RANK = "ranking"
    SELECT = "select"
    CATALOG_SEARCH = "catalog_search"


class Reasons(XRayReasonCode):
    # Filter reasons

    # Price reasons
    PRICE_OK = "PRICE_OK"
    PRICE_TOO_HIGH = "PRICE_TOO_HIGH"
    PRICE_TOO_LOW = "PRICE_TOO_LOW"

    # Rating reasons
    RATING_OK = "RATING_OK"
    LOW_RATING = "LOW_RATING"

    # Category reasons
    CATEGORY_MATCH = "CATEGORY_MATCH"
    CATEGORY_MISMATCH = "CATEGORY_MISMATCH"

    # Review reasons
    REVIEWS_OK = "REVIEWS_OK"
    LOW_REVIEWS = "LOW_REVIEWS"

    # Ranking reasons
    HIGH_RELEVANCE = "HIGH_RELEVANCE"
    MEDIUM_RELEVANCE = "MEDIUM_RELEVANCE"
    LOW_RELEVANCE = "LOW_RELEVANCE"


# =============================================================================
# X-Ray Configuration
# =============================================================================

xray.configure(
    endpoint="http://localhost:8080/api/v1",
    async_send=True,
    fallback="local_file",
    fallback_path="./xray_competitor_fallback",
)

xray.register_pipeline(Pipelines.COMPETITOR_SELECTION, Steps, Reasons)

# Sampling: Always capture rejections, sample 10% of acceptances
sampling = SamplingConfig({
    "rejected": 1.0,
    "accepted": 0.1,
})


# =============================================================================
# Base Product Input
# =============================================================================

BASE_PRODUCT = {
    "asin": "B09V3KXJPB",
    "title": "Apple iPhone 14 Pro Max - 256GB - Deep Purple",
    "category": "Electronics > Cell Phones & Accessories > Cell Phones",
    "price": 1099.00,
    "rating": 4.7,
    "review_count": 12500,
    "attributes": {
        "brand": "Apple",
        "screen_size": "6.7 inches",
        "storage": "256GB",
        "color": "Deep Purple",
        "carrier": "Unlocked",
    }
}


# =============================================================================
# Groq LLM Functions
# =============================================================================

def init_llm():
    api_key = os.environ.get("GROQ_API_KEY")
    if not api_key:
        raise ValueError("GROQ_API_KEY environment variable required")
    return Groq(api_key=api_key)


def call_llm(client: Groq, prompt: str) -> str:
    """Call Groq LLM and return response text."""
    response = client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=[{"role": "user", "content": prompt}],
        temperature=0.7,
    )
    return response.choices[0].message.content.strip()


def parse_json_response(raw: str) -> Any:
    """Parse JSON from LLM response, handling code blocks."""
    text = raw.strip()
    if text.startswith("```"):
        text = text.split("```")[1]
        if text.startswith("json"):
            text = text[4:]
    return json.loads(text.strip())


def generate_keywords(client: Groq, product: Dict) -> List[str]:
    """Generate search keywords from product info."""
    prompt = f"""Generate 5 search keywords for finding competitor products similar to:
Title: {product['title']}
Category: {product['category']}
Brand: {product['attributes'].get('brand', 'Unknown')}

Keywords should not be actual product names, but rather keywords that describe the product.

Return ONLY a JSON array of strings, no explanation. Example: ["keyword1", "keyword2"]"""
    
    raw = call_llm(client, prompt)
    print(f"  [DEBUG] Raw keyword response: {raw[:200]}")
    return parse_json_response(raw)

def search_catalog(client: Groq, keywords: List[str], base_product: Dict) -> List[Dict]:
    """Simulate catalog search by asking LLM to generate competitor products."""
    prompt = f"""You are simulating an Amazon product catalog API.
    
Generate 15 competitor products for: {base_product['title']}
Category: {base_product['category']}
Search keywords: {', '.join(keywords)}

Return ONLY a JSON array of products. Each product must have:
- asin: string (fake Amazon ID FORMAT should be "B0XXXXXXXX", replace X with random integers) (one of the items must have B012345678)
- title: string
- category: string (should vary - some matching, some not)
- price: number (range $200-$2000)
- rating: number (1.0-5.0)
- review_count: number (10-50000)
- brand: string

Include a mix of:
- Direct competitors (Samsung, Google Pixel)
- Somewhat related products (phone cases, tablet, smartwatch)
- Unrelated products (laptop stand, headphones)

This simulates real search results with noise. Return ONLY valid JSON."""

    raw = call_llm(client, prompt)
    print(f"  [DEBUG] Raw catalog response: {raw[:300]}...")
    return parse_json_response(raw)


def generate_filters(client: Groq, base_product: Dict) -> Dict:
    """Generate filter criteria based on base product."""
    prompt = f"""Given this product:
Title: {base_product['title']}
Price: ${base_product['price']}
Category: {base_product['category']}
Rating: {base_product['rating']}

Generate filter criteria for finding similar competitor products.
Return ONLY a JSON object with:
- min_price: number (reasonable lower bound)
- max_price: number (reasonable upper bound)
- min_rating: number (minimum acceptable rating)
- min_reviews: number (minimum review count for credibility)
- target_category_keywords: list of strings (keywords that should appear in category)

Example: {{"min_price": 500, "max_price": 1500, "min_rating": 4.0, "min_reviews": 100, "target_category_keywords": ["phone", "cell", "mobile"]}}"""

    raw = call_llm(client, prompt)
    print(f"  [DEBUG] Raw filter response: {raw[:200]}")
    return parse_json_response(raw)


def rank_candidates(client: Groq, candidates: List[Dict], base_product: Dict) -> List[Dict]:
    """Use LLM to rank candidates by relevance."""
    if not candidates:
        return []
    
    prompt = f"""Rank these products by relevance as competitors to: {base_product['title']}

Products:
{json.dumps(candidates, indent=2)}

For each product, assign a relevance_score (0.0-1.0) and reason.
Return ONLY a JSON array with objects containing:
- asin: string
- relevance_score: number
- reason: string (brief explanation)

Sort by relevance_score descending."""

    raw = call_llm(client, prompt)
    print(f"  [DEBUG] Raw ranking response: {raw[:300]}...")
    rankings = parse_json_response(raw)
    
    # Merge rankings back into candidates
    rank_map = {r["asin"]: r for r in rankings}
    for c in candidates:
        if c["asin"] in rank_map:
            c["relevance_score"] = rank_map[c["asin"]].get("relevance_score", 0.5)
            c["rank_reason"] = rank_map[c["asin"]].get("reason", "")
        else:
            c["relevance_score"] = 0.5
            c["rank_reason"] = "Not ranked"
    
    return sorted(candidates, key=lambda x: x.get("relevance_score", 0), reverse=True)


# =============================================================================
# Filter Functions
# =============================================================================

def apply_filters(
    candidates: List[Dict],
    filters: Dict,
    base_product: Dict,
) -> Tuple[List[Dict], List[Dict]]:
    """Apply filters and return (passed, all_decisions)."""
    passed = []
    decisions = []
    
    for product in candidates:
        item_id = product["asin"]
        price = product.get("price", 0)
        rating = product.get("rating", 0)
        reviews = product.get("review_count", 0)
        category = product.get("category", "").lower()
        
        rejection_reasons = []
        scores = {
            "price": price,
            "rating": rating,
            "review_count": reviews,
        }
        
        # Price filter
        if price < filters["min_price"]:
            rejection_reasons.append(("PRICE_TOO_LOW", f"${price} < ${filters['min_price']}"))
        elif price > filters["max_price"]:
            rejection_reasons.append(("PRICE_TOO_HIGH", f"${price} > ${filters['max_price']}"))
        
        # Rating filter
        if rating < filters["min_rating"]:
            rejection_reasons.append(("LOW_RATING", f"{rating} < {filters['min_rating']}"))
        
        # Review count filter
        if reviews < filters["min_reviews"]:
            rejection_reasons.append(("LOW_REVIEWS", f"{reviews} < {filters['min_reviews']}"))
        
        # Category filter
        category_match = any(kw.lower() in category for kw in filters["target_category_keywords"])
        if not category_match:
            rejection_reasons.append(("CATEGORY_MISMATCH", f"'{category}' missing keywords"))
        
        if rejection_reasons:
            # Use first rejection reason
            reason_code, reason_detail = rejection_reasons[0]
            decisions.append({
                "item_id": item_id,
                "outcome": "rejected",
                "reason_code": reason_code,
                "reason_detail": reason_detail,
                "scores": scores,
                "item_snapshot": {"title": product.get("title", ""), "brand": product.get("brand", "")},
            })
        else:
            passed.append(product)
            decisions.append({
                "item_id": item_id,
                "outcome": "accepted",
                "reason_code": "PASSED_ALL_FILTERS",
                "reason_detail": "Met all filter criteria",
                "scores": scores,
                "item_snapshot": {"title": product.get("title", ""), "brand": product.get("brand", "")},
            })
    
    return passed, decisions


# =============================================================================
# Main Pipeline
# =============================================================================

def run_competitor_selection():
    print(f"\nBase Product: {BASE_PRODUCT['title']}")
    print(f"Price: ${BASE_PRODUCT['price']}")
    print()
    
    client = init_llm()
    
    with xray.trace(
        Pipelines.COMPETITOR_SELECTION,
        input_data=BASE_PRODUCT,
        metadata={"base_asin": BASE_PRODUCT["asin"]},
        tags=["demo", "competitor-selection"],
        sampling_config=sampling,
    ) as trace:
        
        # Step 1: Generate keywords
        with trace.event("generate_keywords", step_type=Steps.KEYWORD_GEN) as e:
            keywords = generate_keywords(client, BASE_PRODUCT)
            e.set_output(keywords, count=len(keywords))
            e.annotate("keywords", keywords)
        
        # Step 2: Search catalog
        with trace.event("search_catalog", step_type=Steps.CATALOG_SEARCH) as e:
            candidates = search_catalog(client, keywords, BASE_PRODUCT)
            e.set_output(candidates, count=len(candidates))
            e.annotate("candidates", candidates)
        
        # Step 3: Generate filters
        with trace.event("generate_filters", step_type=Steps.FILTER) as e:
            filters = generate_filters(client, BASE_PRODUCT)
            e.set_output(filters)
            e.annotate("filters", filters)
        
        # Step 4: Apply filters
        with trace.event("apply_filters", step_type=Steps.FILTER, capture="full") as e:
            e.set_input(candidates, count=len(candidates))
            passed, decisions = apply_filters(candidates, filters, BASE_PRODUCT)
            
            for d in decisions:
                e.record_decision(**d)
            
            e.set_output(passed, count=len(passed))
            e.annotate("rejected_count", len(candidates) - len(passed))
            
            rejected = len(candidates) - len(passed)
        
        # Step 5: Rank candidates
        with trace.event("rank_candidates", step_type=Steps.RANK, capture="full") as e:
            e.set_input(passed, count=len(passed))
            ranked = rank_candidates(client, passed, BASE_PRODUCT)
            
            for product in ranked:
                score = product.get("relevance_score", 0)
                if score >= 0.7:
                    reason_code = Reasons.HIGH_RELEVANCE
                elif score >= 0.4:
                    reason_code = Reasons.MEDIUM_RELEVANCE
                else:
                    reason_code = Reasons.LOW_RELEVANCE
                
                e.record_decision(
                    item_id=product["asin"],
                    outcome="accepted",
                    reason_code=reason_code,
                    reason_detail=product.get("rank_reason", ""),
                    scores={"relevance_score": score},
                )
            
            e.set_output(ranked, count=len(ranked))
        
        # Step 6: Select best match
        with trace.event("select_best", step_type=Steps.SELECT) as e:
            e.set_input(ranked, count=len(ranked))
            
            if ranked:
                best_match = ranked[0]
                e.set_output(best_match)
                e.annotate("selected_asin", best_match["asin"])
                e.annotate("relevance_score", best_match.get("relevance_score", 0))
                print(f"\n  Best Match: {best_match['title']}")
                print(f"     ASIN: {best_match['asin']}")
                print(f"     Price: ${best_match.get('price', 'N/A')}")
                print(f"     Relevance: {best_match.get('relevance_score', 0):.2f}")
            else:
                e.set_output(None)
                e.annotate("no_match", True)
                print("  No suitable competitor found")
                best_match = None
        
        print(f"Pipeline Complete! Trace ID: {trace.trace_id}")
        
        return trace.trace_id, best_match


if __name__ == "__main__":
    trace_id, result = run_competitor_selection()
    
    # Save trace ID for curl commands
    with open("last_trace_id.txt", "w") as f:
        f.write(trace_id)
    
    print(f"\nTrace ID saved to last_trace_id.txt")
    print(f"View trace: curl http://localhost:8080/api/v1/traces/{trace_id}")
