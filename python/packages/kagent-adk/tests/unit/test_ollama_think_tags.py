"""Unit tests for Ollama <think> tag parsing.

This module tests the extraction and parsing of <think></think> tags from
content, used by models that embed reasoning within the content text.

Task: E002
"""

import pytest
from kagent.adk.models._ollama import _parse_think_tags, _convert_ollama_response_to_llm_response


class TestThinkTagParsing:
    """Test <think> tag parsing from content."""

    def test_simple_think_tag_extraction(self):
        """Test extraction of content between <think> tags."""
        content = "<think>Let me analyze this carefully</think>The answer is 42."
        
        thought, remaining = _parse_think_tags(content)
        
        assert thought == "Let me analyze this carefully"
        assert remaining == "The answer is 42."

    def test_multiple_think_tags(self):
        """Test handling of multiple <think> tags."""
        content = "<think>First thought</think>Some text<think>Second thought</think>More text"
        
        thought, remaining = _parse_think_tags(content)
        
        # Should extract all think content
        assert "First thought" in thought
        assert "Second thought" in thought
        assert remaining == "Some textMore text"

    def test_nested_think_tags(self):
        """Test handling of nested tags (should handle gracefully)."""
        content = "<think>Outer<think>Inner</think>Still outer</think>Result"
        
        thought, remaining = _parse_think_tags(content)
        
        # Should extract think content (implementation may vary)
        assert len(thought) > 0
        assert "Result" in remaining

    def test_malformed_think_tags(self):
        """Test handling of malformed tags."""
        # Unclosed tag
        content1 = "<think>Incomplete thought... The answer"
        thought1, remaining1 = _parse_think_tags(content1)
        # Should handle gracefully - either extract partial or leave as-is
        assert remaining1 is not None
        
        # Tag in wrong order
        content2 = "</think>Backwards<think>"
        thought2, remaining2 = _parse_think_tags(content2)
        assert remaining2 is not None

    def test_empty_think_tags(self):
        """Test handling of empty <think> tags."""
        content = "<think></think>The answer"
        
        thought, remaining = _parse_think_tags(content)
        
        assert thought == ""
        assert remaining == "The answer"

    def test_think_tags_with_whitespace(self):
        """Test think tags with various whitespace."""
        content = "<think>  \n  Thoughtful content  \n  </think>  Answer"
        
        thought, remaining = _parse_think_tags(content)
        
        # Should preserve or normalize whitespace appropriately
        assert "Thoughtful content" in thought
        assert "Answer" in remaining

    def test_no_think_tags(self):
        """Test content without any think tags."""
        content = "Just regular content without tags"
        
        thought, remaining = _parse_think_tags(content)
        
        assert thought == ""
        assert remaining == content

    def test_streaming_incremental_think_parsing(self):
        """Test incremental parsing of think tags in streaming."""
        # Chunk 1: Opening tag
        chunk1 = "<think>Starting"
        thought1, remaining1 = _parse_think_tags(chunk1)
        # May not extract until complete tag
        
        # Chunk 2: Middle content
        chunk2 = "<think>Starting to think"
        thought2, remaining2 = _parse_think_tags(chunk2)
        
        # Chunk 3: Closing tag
        chunk3 = "<think>Starting to think carefully</think>Answer"
        thought3, remaining3 = _parse_think_tags(chunk3)
        assert "think carefully" in thought3
        assert remaining3 == "Answer"

    def test_think_tags_in_ollama_response(self):
        """Test think tag parsing integrated with response conversion."""
        ollama_response = {
            "model": "llama3.3",
            "message": {
                "role": "assistant",
                "content": "<think>Let me reason about this</think>The solution is X."
            },
            "done": True
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)

        # Should have reasoning content extracted from think tags
        assert llm_response.custom_metadata is not None
        assert "reasoning_content" in llm_response.custom_metadata
        assert "reason about this" in llm_response.custom_metadata["reasoning_content"]
        
        # Main content should not have think tags
        text_parts = [part for part in llm_response.content.parts if part.text and part.text.strip()]
        if text_parts:
            assert "<think>" not in text_parts[0].text
            assert "solution is X" in text_parts[0].text

    def test_think_tags_with_special_characters(self):
        """Test think tags containing special characters."""
        content = "<think>Math: 2+2=4, Logic: if(x>y) then z</think>Result"
        
        thought, remaining = _parse_think_tags(content)
        
        assert "2+2=4" in thought
        assert "if(x>y)" in thought
        assert remaining == "Result"

    def test_case_sensitive_think_tags(self):
        """Test that tags are case-sensitive."""
        # Wrong case should not be parsed
        content = "<Think>Not parsed</Think><THINK>Also not parsed</THINK>"
        
        thought, remaining = _parse_think_tags(content)
        
        assert thought == ""
        # Should leave incorrect case tags in content
        assert "<Think>" in remaining or remaining == content


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

